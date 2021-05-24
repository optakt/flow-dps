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
	"github.com/onflow/cadence/runtime"

	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type Invoker struct {
	log     zerolog.Logger
	state   dps.State
	vm      *fvm.VirtualMachine
	chainID flow.ChainID
	headers fvm.Blocks
}

func New(log zerolog.Logger, state dps.State, chainID flow.ChainID, headers fvm.Blocks) *Invoker {

	rt := runtime.NewInterpreterRuntime()
	vm := fvm.NewVirtualMachine(rt)

	i := Invoker{
		log:     log,
		state:   state,
		vm:      vm,
		chainID: chainID,
		headers: headers,
	}

	return &i
}

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
	header, err := i.state.Chain().Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve header for height: %w", err)
	}
	commit, err := i.state.Commit().ForHeight(height)
	if err != nil {
		return nil, fmt.Errorf("could not look up height for commit: %w", err)
	}

	// we initialize the virtual machine context with the given block header so
	// that parameters related to the block are available from within the script
	ctx := fvm.NewContext(i.log,
		fvm.WithChain(i.chainID.Chain()),
		fvm.WithBlocks(i.headers),
		fvm.WithBlockHeader(header),
		fvm.WithTransactionProcessors(),
		fvm.WithAccountFreezeAvailable(false),
		fvm.WithServiceAccount(false),
		fvm.WithRestrictedDeployment(true),
		fvm.WithAccountStorageLimit(false),
	)

	// we initialize the view of the execution state on top of our ledger by
	// using the read function at a specific commit
	view := delta.NewView(i.read(commit))

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

func (i *Invoker) read(commit flow.StateCommitment) delta.GetRegisterFunc {

	readCache := make(map[flow.RegisterID]flow.RegisterEntry)
	return func(owner string, controller string, key string) (flow.RegisterValue, error) {

		regID := flow.NewRegisterID(owner, controller, key)
		entry, ok := readCache[regID]
		if ok {
			return entry.Value, nil
		}

		lkey := state.RegisterIDToKey(regID)
		query, err := ledger.NewQuery(commit, []ledger.Key{lkey})
		if err != nil {
			return nil, fmt.Errorf("could not create ledger query: %w", err)
		}

		values, err := i.state.Ledger().Get(query)
		if err != nil {
			return nil, fmt.Errorf("error getting register (%s) value at %x: %w", key, commit, err)
		}
		if len(values) == 0 {
			return nil, nil
		}

		value := values[0]
		readCache[regID] = flow.RegisterEntry{Key: regID, Value: value}

		return value, nil
	}
}
