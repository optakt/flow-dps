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
	"github.com/awfm9/flow-dps/rosetta/retriever"
)

type Validator struct {
	height    dps.Height
	contracts retriever.Contracts
}

func New(height dps.Height, contracts retriever.Contracts) *Validator {

	v := &Validator{
		height:    height,
		contracts: contracts,
	}

	return v
}

// TODO: implement proper validation for network; should depend on the chain
// configuration parameters
// => https://github.com/awfm9/flow-dps/issues/50
func (v *Validator) Network(network identifier.Network) error {

	if network.Blockchain != "flow" {
		return fmt.Errorf("invalid network identifier blockchain (%s)", network.Blockchain)
	}

	if network.Network != "testnet" && network.Network != "mainnet" {
		return fmt.Errorf("invalid network identifier network (%s)", network.Network)
	}

	return nil
}

// TODO: implement validation for block; should distinguish between block we
// don't know yet / haven't seen and blocks that are just mismatched
// => https://github.com/awfm9/flow-dps/issues/51
func (v *Validator) Block(block identifier.Block) error {

	blockID, err := flow.HexStringToIdentifier(block.Hash)
	if err != nil {
		return fmt.Errorf("could not parse block identifier hash: %w", err)
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

	_, err := flow.HexStringToIdentifier(transaction.Hash)
	if err != nil {
		return fmt.Errorf("could not parse transaction identifier hash: %w", err)
	}

	return nil
}

// TODO: implement validation for account; should use address generator to make
// sure that it is a valid address for the configured chain
// => https://github.com/awfm9/flow-dps/issues/53
func (v *Validator) Account(account identifier.Account) error {
	return nil
}

// TODO: implement validation for currency; this should probably be refactored
// after we made tokens configurable
// => https://github.com/awfm9/flow-dps/issues/52
func (v *Validator) Currency(currency identifier.Currency) error {

	if currency.Decimals != 8 {
		return fmt.Errorf("invalid number of decimals for currency identifier (%d)", currency.Decimals)
	}

	_, ok := v.contracts.Token(currency.Symbol)
	if !ok {
		return fmt.Errorf("invalid token symbol for currency identifier (%s)", currency.Symbol)
	}

	return nil
}
