// Copyright 2021 Alvalor S.A.
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

	"github.com/c2h5oh/datasize"
	"github.com/dgraph-io/ristretto"
	"github.com/rs/zerolog"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"

	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"

	"github.com/optakt/flow-dps/models/index"
)

// TODO: Create read cache outside of the `GetRegisterFunc` closure, so we can
// manage the used space:
// => https://github.com/optakt/flow-dps/issues/122

type Invoker struct {
	index index.Reader
	vm    *fvm.VirtualMachine
	cache *ristretto.Cache
}

func New(index index.Reader, options ...func(*Config)) (*Invoker, error) {

	// Initialize the invoker configuration with conservative default values.
	cfg := Config{
		CacheSize: uint64(100 * datasize.MB), // ~100 MB default size
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

// TODO: Find a way to batch up register requests for Cadence execution so we
// don't have to request them one by one over GRPC.
// => https://github.com/optakt/flow-dps/issues/119

func (i *Invoker) Script(height uint64, script []byte, arguments []cadence.Value) (cadence.Value, error) {

	// encode the arguments from cadence values to byte slices
	var args [][]byte
	for _, argument := range arguments {
		arg, err := json.Encode(argument)
		if err != nil {
			return nil, fmt.Errorf("could not encode value: %w", err)
		}
		args = append(args, arg)
	}

	// look up the current block and commit for the block
	header, err := i.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	// we initialize the virtual machine context with the given block header so
	// that parameters related to the block are available from within the script
	ctx := fvm.NewContext(zerolog.Nop(), fvm.WithBlockHeader(header))

	// Initialize the read function. We use a shared cache between all heights
	// here. It's a smart cache, which means that items that are accessed often
	// are more likely to be kept, regardless of height. This allows us to put
	// an upper bound on total cache size while using it for all heights.
	read := readRegister(i.index, i.cache, height)

	// we initialize the view of the execution state on top of our ledger by
	// using the read function at a specific commit
	view := delta.NewView(read)

	// we initialize the procedure using the script bytes and the encoded
	// cadence parameters
	proc := fvm.Script(script).WithArguments(args...)

	// finally, we initialize an empty programs cache
	programs := programs.NewEmptyPrograms()

	// the script procedure is then run using the Flow virtual machine and all
	// of the constructed contextual parameters
	err = i.vm.Run(ctx, proc, view, programs)
	if err != nil {
		return nil, fmt.Errorf("could not run script: %w", err)
	}
	if proc.Err != nil {
		return nil, fmt.Errorf("script execution encountered error: %w", proc.Err)
	}

	return proc.Value, nil
}
