// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package invoker

import (
	"fmt"

	"github.com/dgraph-io/ristretto"
	"github.com/rs/zerolog"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"
	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-dps/models/dps"
)

// Invoker retrieves account information from and executes Cadence scripts against
// the Flow virtual machine.
type Invoker struct {
	index dps.Reader
	vm    VirtualMachine
	cache Cache
}

// New returns a new Invoker with the given configuration.
func New(index dps.Reader, options ...func(*Config)) (*Invoker, error) {

	// Initialize the invoker configuration with conservative default values.
	cfg := Config{
		CacheSize: uint64(100_000_000), // ~100 MB default size
	}

	// Apply the option parameters provided by consumer.
	for _, option := range options {
		option(&cfg)
	}

	// Initialize interpreter and virtual machine for execution.
	rt := fvm.NewInterpreterRuntime()
	vm := fvm.NewVirtualMachine(rt)

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

	return &i, nil
}

// Key returns the public key of the account with the given address.
func (i *Invoker) Key(height uint64, address flow.Address, index int) (*flow.AccountPublicKey, error) {

	// Retrieve the account at the specified block height.
	account, err := i.Account(height, address)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve account: %w", err)
	}

	// Create a key lookup map and check if the requested key exists.
	keys := make(map[int]flow.AccountPublicKey)
	for _, key := range account.Keys {
		keys[key.Index] = key
	}
	key, ok := keys[index]
	if !ok {
		return nil, fmt.Errorf("account key with given index not found")
	}

	// Check if the key is still valid.
	if key.Revoked {
		return nil, fmt.Errorf("account key with given index has been revoked")
	}

	return &key, nil
}

// Account returns the account with the given address.
func (i *Invoker) Account(height uint64, address flow.Address) (*flow.Account, error) {

	// Look up the current block and commit for the block.
	header, err := i.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	ctx := fvm.NewContext(zerolog.Nop(), fvm.WithBlockHeader(header))

	// Initialize the read function. We use a shared cache between all heights
	// here. It's a smart cache, which means that items that are accessed often
	// are more likely to be kept, regardless of height. This allows us to put
	// an upper bound on total cache size while using it for all heights.
	read := readRegister(i.index, i.cache, header.Height)

	// Initialize the view of the execution state on top of the ledger by
	// using the read function at a specific commit.
	view := delta.NewView(read)

	account, err := i.vm.GetAccount(ctx, address, view, programs.NewEmptyPrograms())
	if err != nil {
		return nil, fmt.Errorf("could not get account at height %d: %w", header.Height, err)
	}

	return account, nil
}

// Script executes the given Cadence script and returns its result.
func (i *Invoker) Script(height uint64, script []byte, arguments []cadence.Value) (cadence.Value, error) {

	// Encode the arguments from Cadence values to byte slices.
	var args [][]byte
	for _, argument := range arguments {
		arg, err := json.Encode(argument)
		if err != nil {
			return nil, fmt.Errorf("could not encode value: %w", err)
		}
		args = append(args, arg)
	}

	// Look up the current block and commit for the block.
	header, err := i.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	// Initialize the virtual machine context with the given block header so
	// that parameters related to the block are available from within the script.
	ctx := fvm.NewContext(zerolog.Nop(), fvm.WithBlockHeader(header))

	// Initialize the read function. We use a shared cache between all heights
	// here. It's a smart cache, which means that items that are accessed often
	// are more likely to be kept, regardless of height. This allows us to put
	// an upper bound on total cache size while using it for all heights.
	read := readRegister(i.index, i.cache, height)

	// Initialize the view of the execution state on top of the ledger by
	// using the read function at a specific commit.
	view := delta.NewView(read)

	// Initialize the procedure using the script bytes and the encoded
	// Cadence parameters.
	proc := fvm.Script(script).WithArguments(args...)

	// Finally, we initialize an empty programs cache.
	programs := programs.NewEmptyPrograms()

	// The script procedure is then run using the Flow virtual machine and all
	// the constructed contextual parameters.
	err = i.vm.Run(ctx, proc, view, programs)
	if err != nil {
		return nil, fmt.Errorf("could not run script: %w", err)
	}
	if proc.Err != nil {
		return nil, fmt.Errorf("script execution encountered error: %w", proc.Err)
	}

	return proc.Value, nil
}
