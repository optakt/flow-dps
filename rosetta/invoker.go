package rosetta

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"

	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/rest"
)

type Invoker struct {
	log   zerolog.Logger
	state rest.State
	vm    *fvm.VirtualMachine
}

func NewInvoker(log zerolog.Logger, state rest.State) (*Invoker, error) {

	rt := runtime.NewInterpreterRuntime()
	vm := fvm.NewVirtualMachine(rt)

	i := &Invoker{
		log:   log,
		state: state,
		vm:    vm,
	}

	return i, nil
}

func (i *Invoker) RunScript(commit flow.StateCommitment, script []byte, arguments ...[]byte) ([]byte, error) {
	ctx := fvm.NewContext(i.log, fvm.WithTransactionProcessors())
	view := delta.NewView(i.read(commit))
	proc := fvm.Script(script).WithArguments(arguments...)
	programs := programs.NewEmptyPrograms()
	err := i.vm.Run(ctx, proc, view, programs)
	if err != nil {
		return nil, fmt.Errorf("could not run script: %w", err)
	}
	if proc.Err != nil {
		return nil, fmt.Errorf("script execution encountered error: %w", err)
	}
	value, err := json.Encode(proc.Value)
	if err != nil {
		return nil, fmt.Errorf("could not encode result: %w", err)
	}
	return value, nil
}

func (i *Invoker) read(commit flow.StateCommitment) delta.GetRegisterFunc {
	return func(owner string, controller string, key string) (flow.RegisterValue, error) {
		regID := flow.NewRegisterID(owner, controller, key)
		lkey := state.RegisterIDToKey(regID)
		query, err := ledger.NewQuery(commit, []ledger.Key{lkey})
		if err != nil {
			return nil, fmt.Errorf("could not create ledger query: %w", err)
		}
		values, err := i.state.Ledger().Get(query)
		if err != nil {
			return nil, fmt.Errorf("could not get ledger values: %w", err)
		}
		return values[0], nil
	}
}
