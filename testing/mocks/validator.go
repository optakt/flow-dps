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
	AccountFunc     func(addressQualifier identifier.Account) error
	BlockFunc       func(blockQualifier identifier.Block) (identifier.Block, error)
	TransactionFunc func(transactionQualifier identifier.Transaction) error
	CurrencyFunc    func(currencyQualifiers identifier.Currency) (identifier.Currency, error)
}

func BaselineValidator(t *testing.T) *Validator {
	t.Helper()

	v := Validator{
		AccountFunc: func(addressQualifier identifier.Account) error {
			return nil
		},
		BlockFunc: func(blockQualifier identifier.Block) (identifier.Block, error) {
			return GenericBlockQualifier, nil
		},
		TransactionFunc: func(transactionQualifier identifier.Transaction) error {
			return nil
		},
		CurrencyFunc: func(currencyQualifier identifier.Currency) (identifier.Currency, error) {
			return GenericCurrency, nil
		},
	}

	return &v
}

func (v *Validator) Account(addressQualifier identifier.Account) error {
	return v.AccountFunc(addressQualifier)
}

func (v *Validator) Block(blockQualifier identifier.Block) (identifier.Block, error) {
	return v.BlockFunc(blockQualifier)
}

func (v *Validator) Transaction(transactionQualifier identifier.Transaction) error {
	return v.TransactionFunc(transactionQualifier)
}

func (v *Validator) Currency(currencyQualifier identifier.Currency) (identifier.Currency, error) {
	return v.CurrencyFunc(currencyQualifier)
}
