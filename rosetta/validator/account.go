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

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

// Account validates the given account identifier, and if successful, returns a matching Flow Address.
func (v *Validator) Account(account identifier.Account) (flow.Address, error) {

	// Parse the address; the length was already validated, but it's still
	// possible that the characters are not valid hex encoding.
	bytes, err := hex.DecodeString(account.Address)
	if err != nil {
		return flow.EmptyAddress, failure.InvalidAccount{
			Address:     account.Address,
			Description: failure.NewDescription(addressInvalid),
		}
	}

	// We use the Flow chain address generator to check if the converted address
	// is valid.
	var address flow.Address
	copy(address[:], bytes)
	ok := v.params.ChainID.Chain().IsValid(address)
	if !ok {
		return flow.EmptyAddress, failure.InvalidAccount{
			Address: account.Address,
			Description: failure.NewDescription(addressMisconfigured,
				failure.WithString("active_chain", v.params.ChainID.String()),
			),
		}
	}

	return address, nil
}
