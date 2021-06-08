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

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

func (v *Validator) Currency(currency *identifier.Currency) error {

	// We already checked the token symbol is given, so this merely checks if
	// the token has been configured yet.
	_, ok := v.params.Tokens[currency.Symbol]
	if !ok {
		return failure.UnknownCurrency{
			Symbol:   currency.Symbol,
			Decimals: currency.Decimals,
			Message:  "currency not currently configured",
		}
	}

	// If the token is known, the decimals should always be 8, as we always use
	// `UFix64` for tokens on Flow.
	if currency.Decimals != 0 && currency.Decimals != dps.FlowDecimals {
		return failure.InvalidCurrency{
			Symbol:   currency.Symbol,
			Decimals: currency.Decimals,
			Message:  fmt.Sprintf("invalid number of decimals (have: %d, want: %d)", currency.Decimals, dps.FlowDecimals),
		}
	}

	// At this point decimals are either 8 or empty, so set it.
	currency.Decimals = 8

	return nil
}
