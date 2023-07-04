package mocks

import (
	"testing"

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go/model/flow"
)

type Invoker struct {
	KeyFunc     func(height uint64, address flow.Address, index int) (*flow.AccountPublicKey, error)
	AccountFunc func(height uint64, address flow.Address) (*flow.Account, error)
	ScriptFunc  func(height uint64, script []byte, parameters [][]byte) ([]byte, error)
}

func BaselineInvoker(t *testing.T) *Invoker {
	t.Helper()

	i := Invoker{
		KeyFunc: func(height uint64, address flow.Address, index int) (*flow.AccountPublicKey, error) {
			return &GenericAccount.Keys[0], nil
		},
		AccountFunc: func(height uint64, address flow.Address) (*flow.Account, error) {
			return &GenericAccount, nil
		},
		ScriptFunc: func(height uint64, script []byte, parameters [][]byte) ([]byte, error) {
			return json.MustEncode(GenericAmount(0)), nil
		},
	}

	return &i
}

func (i *Invoker) Key(height uint64, address flow.Address, index int) (*flow.AccountPublicKey, error) {
	return i.KeyFunc(height, address, index)
}

func (i *Invoker) Account(height uint64, address flow.Address) (*flow.Account, error) {
	return i.AccountFunc(height, address)
}

func (i *Invoker) Script(height uint64, script []byte, parameters [][]byte) ([]byte, error) {
	return i.ScriptFunc(height, script, parameters)
}
