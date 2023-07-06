package mocks

import (
	"context"
	"testing"

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go/model/flow"
)

type Invoker struct {
	AccountFunc func(ctx context.Context, height uint64, address flow.Address) (*flow.Account, error)
	ScriptFunc  func(ctx context.Context, height uint64, script []byte, parameters [][]byte) ([]byte, error)
}

func BaselineInvoker(t *testing.T) *Invoker {
	t.Helper()

	i := Invoker{
		AccountFunc: func(ctx context.Context, height uint64, address flow.Address) (*flow.Account, error) {
			return &GenericAccount, nil
		},
		ScriptFunc: func(ctx context.Context, height uint64, script []byte, parameters [][]byte) ([]byte, error) {
			return json.MustEncode(GenericAmount(0)), nil
		},
	}

	return &i
}

func (i *Invoker) Account(ctx context.Context, height uint64, address flow.Address) (*flow.Account, error) {
	return i.AccountFunc(ctx, height, address)
}

func (i *Invoker) Script(ctx context.Context, height uint64, script []byte, parameters [][]byte) ([]byte, error) {
	return i.ScriptFunc(ctx, height, script, parameters)
}
