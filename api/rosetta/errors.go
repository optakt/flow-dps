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

package rosetta

import (
	"fmt"

	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/meta"
)

type Error struct {
	meta.ErrorDefinition
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

func Internal(err error) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInternal,
		Description:     err.Error(),
		Details:         nil,
	}
}

func InvalidFormat(message string, args ...interface{}) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidFormat,
		Description:     fmt.Sprintf(message, args...),
		Details:         nil,
	}
}

func InvalidNetwork(fail failure.InvalidNetwork) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidNetwork,
		Description:     fail.Message,
		Details: map[string]interface{}{
			"blockchain": fail.Blockchain,
			"network":    fail.Network,
		},
	}
}

func InvalidBlock(fail failure.InvalidBlock) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidBlock,
		Description:     fail.Message,
		Details: map[string]interface{}{
			"height": fail.Height,
			"block":  fail.BlockID.String(),
		},
	}
}

func InvalidAccount(fail failure.InvalidAccount) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidAccount,
		Description:     fail.Message,
		Details: map[string]interface{}{
			"address": fail.Address.String(),
		},
	}
}

func InvalidCurrency(fail failure.InvalidCurrency) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidCurrency,
		Description:     fail.Message,
		Details: map[string]interface{}{
			"symbol":   fail.Symbol,
			"decimals": fail.Decimals,
		},
	}
}

func UnknownBlock(fail failure.UnknownBlock) Error {
	return Error{
		ErrorDefinition: configuration.ErrorUnknownBlock,
		Description:     fail.Message,
		Details: map[string]interface{}{
			"height": fail.Height,
			"block":  fail.BlockID.String(),
		},
	}
}

func UnknownCurrency(fail failure.UnknownCurrency) Error {
	return Error{
		ErrorDefinition: configuration.ErrorUnknownCurrency,
		Description:     fail.Message,
		Details: map[string]interface{}{
			"symbol":   fail.Symbol,
			"decimals": fail.Decimals,
		},
	}
}
