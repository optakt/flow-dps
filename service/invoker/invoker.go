package invoker

import (
	"context"
	"fmt"

	"github.com/dgraph-io/ristretto"
	"github.com/rs/zerolog"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-archive/util"
	"github.com/onflow/flow-go/engine/execution/computation"
	"github.com/onflow/flow-go/engine/execution/computation/query"
	"github.com/onflow/flow-go/fvm"
	reusableRuntime "github.com/onflow/flow-go/fvm/runtime"
	"github.com/onflow/flow-go/fvm/storage/derived"
	"github.com/onflow/flow-go/fvm/storage/snapshot"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/metrics"
)

// Invoker retrieves account information from and executes Cadence scripts against
// the Flow virtual machine.
type Invoker struct {
	index         archive.Reader
	queryExecutor *query.QueryExecutor
	cache         Cache
	*Blocks
}

// New returns a new Invoker with the given configuration.
func New(
	index archive.Reader,
	cfg Config,
) (*Invoker, error) {

	// Initialize the Ristretto cache with the size limit. Ristretto recommends
	// keeping ten times as many counters as items in the cache when full.
	// Assuming an average item size of 1 kilobyte, this is what we get.
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(cfg.CacheSize) / 1000 * 10,
		MaxCost:     int64(cfg.CacheSize),
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize cache: %w", err)
	}

	blocks := NewBlocks(index)

	// This is copied code from flow-go engine/execution/computation/manager.go
	// Ideally the same code would be used, but that requires flow-go changes.
	// TODO: reuse code from flow-go
	var vm fvm.VM
	if cfg.NewCustomVirtualMachine != nil {
		vm = cfg.NewCustomVirtualMachine()
	} else {
		vm = fvm.NewVirtualMachine()
	}

	chainID := cfg.ChainID
	fvmOptions := []fvm.Option{
		fvm.WithReusableCadenceRuntimePool(
			reusableRuntime.NewReusableCadenceRuntimePool(
				computation.ReusableCadenceRuntimePoolSize,
				runtime.Config{
					TracingEnabled:        cfg.CadenceTracing,
					AccountLinkingEnabled: true,
					// Attachments are enabled everywhere except for Mainnet
					AttachmentsEnabled: chainID != flow.Mainnet,
					// Capability Controllers are enabled everywhere except for Mainnet
					CapabilityControllersEnabled: chainID != flow.Mainnet,
				},
			),
		),
		fvm.WithBlocks(blocks),
	}

	vmCtx := fvm.NewContext(fvmOptions...)
	derivedChainData, err := derived.NewDerivedChainData(cfg.DerivedDataCacheSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create derived data cache: %w", err)
	}

	queryExecutor := query.NewQueryExecutor(
		cfg.QueryConfig,
		zerolog.Nop(),            // TODO: add logger
		&metrics.NoopCollector{}, // TODO: add metrics
		vm,
		vmCtx,
		derivedChainData,
	)

	return &Invoker{
		Blocks:        blocks,
		index:         index,
		cache:         cache,
		queryExecutor: queryExecutor,
	}, nil
}

// Key returns the public key of the account with the given address.
func (i *Invoker) Key(height uint64, address flow.Address, index int) (*flow.AccountPublicKey, error) {
	// Retrieve the account at the specified block height.
	account, err := i.Account(height, address)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve account: %w", err)
	}

	for _, key := range account.Keys {
		if key.Index != index {
			continue
		}

		if key.Revoked {
			return nil, fmt.Errorf("account key with given index has been revoked")
		}
		return &key, nil
	}

	return nil, fmt.Errorf("account key with given index not found")
}

// Account returns the account with the given address.
func (i *Invoker) Account(height uint64, address flow.Address) (*flow.Account, error) {
	err := util.ValidateHeightDataAvailable(i.index, height)
	if err != nil {
		return nil, err
	}
	// Look up the current block and commit for the block.
	header, err := i.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	return i.queryExecutor.GetAccount(
		context.Background(), // TODO: add context
		address,
		header,
		i.storageSnapshot(height),
	)
}

// Script executes the given Cadence script and returns its result.
func (i *Invoker) Script(height uint64, script []byte, args [][]byte) ([]byte, error) {
	err := util.ValidateHeightDataAvailable(i.index, height)
	if err != nil {
		return nil, err
	}
	// Look up the current block and commit for the block.
	header, err := i.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	return i.queryExecutor.ExecuteScript(
		context.Background(), // TODO: add context
		script,
		args,
		header,
		i.storageSnapshot(height),
	)
}

func (i *Invoker) storageSnapshot(height uint64) snapshot.StorageSnapshot {
	// Initialize the storage snapshot. We use a shared cache between all
	// heights here. It's a smart cache, which means that items that are
	// accessed often are more likely to be kept, regardless of height. This
	// allows us to put an upper bound on total cache size while using it for
	// all heights.
	return snapshot.NewReadFuncStorageSnapshot(
		readRegister(i.index, i.cache, height))
}

func readRegister(
	index archive.Reader,
	cache Cache,
	height uint64,
) func(flow.RegisterID) (flow.RegisterValue, error) {
	return func(regID flow.RegisterID) (flow.RegisterValue, error) {
		cacheKey := fmt.Sprintf("%d/%s", height, regID)
		cacheValue, ok := cache.Get(cacheKey)
		if ok {
			return cacheValue.(flow.RegisterValue), nil
		}

		values, err := index.Values(height, flow.RegisterIDs{regID})
		if err != nil {
			return nil, fmt.Errorf("could not read register: %w", err)
		}
		if len(values) != 1 {
			return nil, fmt.Errorf("wrong number of register values: %d", len(values))
		}

		value := values[0]
		_ = cache.Set(cacheKey, value, int64(len(value)))

		return value, nil
	}
}
