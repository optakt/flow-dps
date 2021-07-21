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

package mocks

import (
	"testing"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"
)

type Invoker struct {
	ScriptFunc     func(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error)
	GetAccountFunc func(address flow.Address, header *flow.Header) (*flow.Account, error)
}

func BaselineInvoker(t *testing.T) *Invoker {
	t.Helper()

	i := Invoker{
		ScriptFunc: func(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error) {
			return GenericAmount(0), nil
		},
		GetAccountFunc: func(address flow.Address, header *flow.Header) (*flow.Account, error) {
			return &GenericAccount, nil
		},
	}

	return &i
}

func (i *Invoker) Script(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error) {
	return i.ScriptFunc(height, script, parameters)
}

func (i *Invoker) GetAccount(address flow.Address, header *flow.Header) (*flow.Account, error) {
	return i.GetAccountFunc(address, header)
}
