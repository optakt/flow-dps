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

package validator

import (
	"encoding/hex"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/rosetta/errors"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

func (v *Validator) Account(account identifier.Account) error {

	// Parse the address; the length was already validated, but it's still
	// possible that the characters are not valid hex encoding.
	bytes, err := hex.DecodeString(account.Address)
	if err != nil {
		return errors.InvalidAccount{
			Description: "account address is not a valid hex-encoded string",
			Details: []errors.Detail{
				errors.WithString("address", account.Address),
				errors.WithString("chain", v.params.ChainID.String()),
			},
		}
	}

	// We use the Flow chain address generator to check if the converted address
	// is valid.
	var address flow.Address
	copy(address[:], bytes)
	ok := v.params.ChainID.Chain().IsValid(address)
	if !ok {
		return errors.InvalidAccount{
			Description: "account address is not valid for configured chain",
			Details: []errors.Detail{
				errors.WithString("address", account.Address),
				errors.WithString("chain", v.params.ChainID.String()),
			},
		}
	}

	return nil
}
