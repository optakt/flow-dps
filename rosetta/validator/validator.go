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
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
	"github.com/awfm9/flow-dps/models/identifier"
)

type Validator struct {
	height dps.Height
}

func New(height dps.Height) *Validator {

	v := &Validator{
		height: height,
	}

	return v
}

func (v *Validator) Network(network identifier.Network) error {

	if network.Blockchain != "flow" {
		return fmt.Errorf("invalid network identifier blockchain (%s)", network.Blockchain)
	}

	if network.Network != "testnet" && network.Network != "mainnet" {
		return fmt.Errorf("invalid network identifier network (%s)", network.Network)
	}

	return nil
}

func (v *Validator) Block(block identifier.Block) error {

	blockID, err := flow.HexStringToIdentifier(block.Hash)
	if err != nil {
		return fmt.Errorf("could not parse block hash: %w", err)
	}

	height, err := v.height.ForBlock(blockID)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not validate block identifier: %w", err)
	}

	if height != block.Index {
		return fmt.Errorf("could not match block identifier index to height (%d != %d)", block.Index, height)
	}

	return nil
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
