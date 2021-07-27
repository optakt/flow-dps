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

	"github.com/optakt/flow-dps/rosetta/identifier"
)

type Validator struct {
	AccountFunc     func(rosAddress identifier.Account) error
	BlockFunc       func(rosBlockID identifier.Block) (identifier.Block, error)
	TransactionFunc func(rosTxID identifier.Transaction) error
	CurrencyFunc    func(rosCurrencies identifier.Currency) (identifier.Currency, error)
}

func BaselineValidator(t *testing.T) *Validator {
	t.Helper()

	v := Validator{
		AccountFunc: func(rosAddress identifier.Account) error {
			return nil
		},
		BlockFunc: func(rosBlockID identifier.Block) (identifier.Block, error) {
			return GenericBlockQualifier, nil
		},
		TransactionFunc: func(rosTxID identifier.Transaction) error {
			return nil
		},
		CurrencyFunc: func(rosCurrency identifier.Currency) (identifier.Currency, error) {
			return GenericCurrency, nil
		},
	}

	return &v
}

func (v *Validator) Account(rosAddress identifier.Account) error {
	return v.AccountFunc(rosAddress)
}

func (v *Validator) Block(rosBlockID identifier.Block) (identifier.Block, error) {
	return v.BlockFunc(rosBlockID)
}

func (v *Validator) Transaction(rosTxID identifier.Transaction) error {
	return v.TransactionFunc(rosTxID)
}

func (v *Validator) Currency(rosCurrency identifier.Currency) (identifier.Currency, error) {
	return v.CurrencyFunc(rosCurrency)
}
