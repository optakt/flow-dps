package rosetta

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"

	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/model/dps"
)

type Invoker struct {
	log     zerolog.Logger
	state   dps.State
	vm      *fvm.VirtualMachine
	options []fvm.Option
}

func NewInvoker(log zerolog.Logger, state dps.State, options ...fvm.Option) (*Invoker, error) {

	// TODO: add current block for script parameter availability

	rt := runtime.NewInterpreterRuntime()
	vm := fvm.NewVirtualMachine(rt)

	i := &Invoker{
		log:     log,
		state:   state,
		vm:      vm,
		options: options,
	}

	return i, nil
}

func (i *Invoker) RunScript(commit flow.StateCommitment, script []byte, arguments ...[]byte) (cadence.Value, error) {
	ctx := fvm.NewContext(i.log, i.options...)
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
	_ = uint64(proc.Value.ToGoValue().(cadence.UFix64))
	return proc.Value, nil
}

func (i *Invoker) read(commit flow.StateCommitment) delta.GetRegisterFunc {

	readCache := make(map[flow.RegisterID]flow.RegisterEntry)
	return func(owner, controller, key string) (flow.RegisterValue, error) {

		regID := flow.NewRegisterID(owner, controller, key)
		if value, ok := readCache[regID]; ok {
			return value.Value, nil
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
		readCache[regID] = flow.RegisterEntry{Key: regID, Value: values[0]}

		return values[0], nil
	}
}
