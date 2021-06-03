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

	"github.com/rs/zerolog"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"

	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"

	"github.com/optakt/flow-dps/models/dps"
)

// TODO: Create read cache outside of the `GetRegisterFunc` closure, so we can
// manage the space taken up by it.

type Invoker struct {
	index dps.IndexReader
	vm    *fvm.VirtualMachine
	reads map[uint64]delta.GetRegisterFunc
}

func New(index dps.IndexReader) *Invoker {

	rt := fvm.NewInterpreterRuntime()
	vm := fvm.NewVirtualMachine(rt)

	i := Invoker{
		index: index,
		vm:    vm,
		reads: make(map[uint64]delta.GetRegisterFunc),
	}

	return &i
}

// TODO: Find a way to batch up register requests for Cadence execution so we
// don't have to request them one by one over GRPC.

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
		return nil, fmt.Errorf("could not look up header and commit: %w", err)
	}

	// we initialize the virtual machine context with the given block header so
	// that parameters related to the block are available from within the script
	ctx := fvm.NewContext(zerolog.Nop(), fvm.WithBlockHeader(header))

	// get the read function, if it was already initialized, to use the cache
	read, ok := i.reads[height]
	if !ok {
		read = readRegister(i.index, height)
		i.reads[height] = read
	}

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
