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
	"github.com/optakt/flow-dps/models/index"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

type Validator struct {
	params dps.Params
	index  index.Reader
}

func New(params dps.Params, index index.Reader) *Validator {

	v := &Validator{
		params: params,
		index:  index,
	}

	return v
}

func (v *Validator) Transaction(transaction identifier.Transaction) error {
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
