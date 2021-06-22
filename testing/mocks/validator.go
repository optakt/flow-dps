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
	"github.com/optakt/flow-dps/rosetta/identifier"
)

type Validator struct {
	AccountFunc     func(address identifier.Account) error
	BlockFunc       func(block identifier.Block) (identifier.Block, error)
	TransactionFunc func(transaction identifier.Transaction) error
	CurrencyFunc    func(currency identifier.Currency) (identifier.Currency, error)
}

func (v *Validator) Account(address identifier.Account) error {
	return v.AccountFunc(address)
}

func (v *Validator) Block(block identifier.Block) (identifier.Block, error) {
	return v.BlockFunc(block)
}

func (v *Validator) Transaction(transaction identifier.Transaction) error {
	return v.TransactionFunc(transaction)
}

func (v *Validator) Currency(currency identifier.Currency) (identifier.Currency, error) {
	return v.CurrencyFunc(currency)
}
