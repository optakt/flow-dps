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

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/rosetta/identifier"
)

type Validator struct {
	AccountFunc     func(rosAccountID identifier.Account) (flow.Address, error)
	BlockFunc       func(rosBlockID identifier.Block) (uint64, flow.Identifier, error)
	TransactionFunc func(rosTxID identifier.Transaction) (flow.Identifier, error)
	CurrencyFunc    func(rosCurrencies identifier.Currency) (string, uint, error)
}

func BaselineValidator(t *testing.T) *Validator {
	t.Helper()

	v := Validator{
		AccountFunc: func(rosAccountID identifier.Account) (flow.Address, error) {
			return GenericAddress(0), nil
		},
		BlockFunc: func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			return GenericHeader.Height, GenericHeader.ID(), nil
		},
		TransactionFunc: func(rosTxID identifier.Transaction) (flow.Identifier, error) {
			return GenericTransaction(0).ID(), nil
		},
		CurrencyFunc: func(rosCurrency identifier.Currency) (string, uint, error) {
			return GenericCurrency.Symbol, GenericCurrency.Decimals, nil
		},
	}

	return &v
}

func (v *Validator) Account(rosAccountID identifier.Account) (flow.Address, error) {
	return v.AccountFunc(rosAccountID)
}

func (v *Validator) Block(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
	return v.BlockFunc(rosBlockID)
}

func (v *Validator) Transaction(rosTxID identifier.Transaction) (flow.Identifier, error) {
	return v.TransactionFunc(rosTxID)
}

func (v *Validator) Currency(rosCurrency identifier.Currency) (string, uint, error) {
	return v.CurrencyFunc(rosCurrency)
}
