package invoker

import (
	"fmt"

	"github.com/dgraph-io/ristretto"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-archive/util"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/environment"
	"github.com/onflow/flow-go/fvm/errors"
	"github.com/onflow/flow-go/fvm/storage/snapshot"
	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-archive/models/archive"
)

// Invoker retrieves account information from and executes Cadence scripts against
// the Flow virtual machine.
type Invoker struct {
	index archive.Reader
	vm    fvm.VM
	vmCtx fvm.Context
	cache Cache
}

// ensure Invoker implemented this interface
var _ environment.Blocks = (*Invoker)(nil)

// New returns a new Invoker with the given configuration.
func New(index archive.Reader, options ...func(*Config)) (*Invoker, error) {

	// Initialize the invoker configuration with conservative default values.
	cfg := Config{
		CacheSize: uint64(100_000_000), // ~100 MB default size
	}

	// Apply the option parameters provided by consumer.
	for _, option := range options {
		option(&cfg)
	}

	// Initialize interpreter and virtual machine for execution.
	vm := fvm.NewVirtualMachine()

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

	i := Invoker{
		index: index,
		vm:    vm,
		cache: cache,
	}

	vmCtx := fvm.NewContext(fvm.WithBlocks(&i))

	i.vmCtx = vmCtx

	return &i, nil
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

	ctx := fvm.NewContextFromParent(i.vmCtx, fvm.WithBlockHeader(header))

	account, err := i.vm.GetAccount(ctx, address, i.storageSnapshot(height))
	if err != nil {
		return nil, fmt.Errorf("could not get account at height %d: %w", header.Height, err)
	}

	return account, nil
}

// Script executes the given Cadence script and returns its result.
func (i *Invoker) Script(height uint64, script []byte, args [][]byte) (cadence.Value, error) {
	err := util.ValidateHeightDataAvailable(i.index, height)
	if err != nil {
		return nil, err
	}
	// Look up the current block and commit for the block.
	header, err := i.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	ctx := fvm.NewContextFromParent(i.vmCtx, fvm.WithBlockHeader(header))

	// Initialize the procedure using the script bytes and the encoded
	// Cadence parameters.
	proc := fvm.Script(script).WithArguments(args...)

	// The script procedure is then run using the Flow virtual machine and all
	// the constructed contextual parameters.
	_, output, err := i.vm.Run(ctx, proc, i.storageSnapshot(height))
	if err != nil {
		return nil, fmt.Errorf("could not run script: %w", err)
	}
	if output.Err != nil {
		return nil, fmt.Errorf("script execution encountered error: %w", output.Err)
	}

	return output.Value, nil
}

// ByHeightFrom implements the fvm/env/blocks interface
func (i *Invoker) ByHeightFrom(height uint64, header *flow.Header) (*flow.Header, error) {
	if header.Height == height {
		return header, nil
	}

	if height > header.Height {
		minHeight, err := i.index.First()
		if err != nil {
			return nil, fmt.Errorf("could not find first indexed height: %w", err)
		}
		// imitate flow-go error format
		err = errors.NewValueErrorf(fmt.Sprint(height),
			"requested height (%d) is not in the range(%d, %d)", height, minHeight, header.Height)
		return nil, fmt.Errorf("cannot retrieve block parent: %w", err)
	}
	// find the block in storage as all of them are guaranteed to be finalized
	return i.index.Header(height)
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
