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

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

type Validator struct {
	params dps.Params
}

func New(params dps.Params) *Validator {

	v := &Validator{
		params: params,
	}

	return v
}

func (v *Validator) Network(network identifier.Network) error {

	// Rosetta uses `Blockchain` to identify the blockchain type, as opposed to
	// other blockchains such as Bitcoin or Ethereum. Flow, however, internally
	// uses `ChainID` and `Chain` to describe different instances of Flow. We
	// thus use the nomencloture of `flow` being the `Blockchain` and
	// `flow-testnet` and `flow-mainnet` being the `Chains`.
	// In other words, `Blockchain` is the same between Rosetta and Flow, but
	// the Rosetta `Network` corresponds to a Flow `ChainID`.

	if network.Blockchain != dps.FlowBlockchain {
		return fmt.Errorf("invalid blockchain in network identifier (blockchain: %s, expected: %s)", network.Blockchain, dps.FlowBlockchain)
	}

	if flow.ChainID(network.Network) != v.params.ChainID {
		return fmt.Errorf("invalid network in network identifier (network: %s, expected: %s)", network.Network, v.params.ChainID)
	}

	return nil
}

// TODO: implement validation for block; should distinguish between block we
// don't know yet / haven't seen and blocks that are just mismatched
// => https://github.com/optakt/flow-dps/issues/51
func (v *Validator) Block(block identifier.Block) error {

	return nil
}

func (v *Validator) Transaction(transaction identifier.Transaction) error {

	// We parse the transaction hash explicitely to see if it has a valid format.
	_, err := flow.HexStringToIdentifier(transaction.Hash)
	if err != nil {
		return fmt.Errorf("could not parse transaction identifier hash: %w", err)
	}

	return nil
}

func (v *Validator) Account(account identifier.Account) error {

	// We use the Flow chain address generator to check if the converted address
	// is valid.
	address := flow.HexToAddress(account.Address)
	ok := v.params.ChainID.Chain().IsValid(address)
	if !ok {
		return fmt.Errorf("invalid address for configured chain (address: %s)", account.Address)
	}

	return nil
}

func (v *Validator) Currency(currency identifier.Currency) error {

	// Any token on the Flow network uses the `UFix64` type, which always has
	// the same number of decimals (8).
	if currency.Decimals != dps.FlowDecimals {
		return fmt.Errorf("invalid number of decimals for currency identifier (decimals: %d, expected: %d)", currency.Decimals, dps.FlowDecimals)
	}

	// Additionally, the token should be knows for the chain that this DPS node
	// is configured for.
	_, ok := v.params.Tokens[currency.Symbol]
	if !ok {
		return fmt.Errorf("invalid token symbol for currency identifier (symbol: %s)", currency.Symbol)
	}

	return nil
}
