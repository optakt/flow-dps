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

package validator

import (
	"fmt"

	"github.com/awfm9/flow-dps/models/identifier"
)

type Validator struct {
}

func New() *Validator {

	v := &Validator{}

	return v
}

func (v *Validator) Network(network identifier.Network) error {
	// TODO: implement validation for network
	// => https://github.com/awfm9/flow-dps/issues/50
	return fmt.Errorf("not implemented")
}

func (v *Validator) Block(block identifier.Block) error {
	// TODO: implement validation for block
	// => https://github.com/awfm9/flow-dps/issues/51
	return fmt.Errorf("not implemented")
}

func (v *Validator) Transaction(transaction identifier.Transaction) error {
	// TODO: implement validation for transaction
	// => https://github.com/awfm9/flow-dps/issues/54
	return fmt.Errorf("not implemented")
}

func (v *Validator) Account(account identifier.Account) error {
	// TODO: implement validation for account
	// => https://github.com/awfm9/flow-dps/issues/53
	return fmt.Errorf("not implemented")
}

func (v *Validator) Currency(currency identifier.Currency) error {
	// TODO: implement validation for currency
	// => https://github.com/awfm9/flow-dps/issues/52
	return fmt.Errorf("not implemented")
}
