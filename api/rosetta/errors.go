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

package rosetta

import (
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/meta"
)

// Error represents an error as defined by the Rosetta API specification. It
// contains an error definition, which has an error code, error message and
// retriable flag that never change, as well as a description and a list of
// details to provide more granular error information.
// See: https://www.rosetta-api.org/docs/api_objects.html#error
type Error struct {
	meta.ErrorDefinition
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

func RosettaError(definition meta.ErrorDefinition, details failure.Details, fields ...Field) Error {
	details := make(map[string]interface{})
	for key, val := range details.Fields {
		details[key] = val
	}
	e := Error{
		ErrorDefinition: definition,
		Description:     description,
		Details:         details,
	}
	return e
}

func Internal(err error) Error {
	return RosettaError(
		configuration.ErrorInternal,
		"",
		failure.WithError(err),
	)
}

func InvalidEncoding(err error) Error {
	return RosettaError(
		configuration.ErrorInvalidEncoding,
		"body does not have valid JSON",
		failure.WithError(err),
	)
}

func InvalidFormat(description string, fields ...failure.Field) Error {
	return RosettaError(
		configuration.ErrorInvalidFormat,
		description,
		fields...,
	)
}

func InvalidNetwork(fail failure.InvalidNetwork) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidNetwork,
		Description:     fail.Description,
		Details: map[string]interface{}{
			"blockchain": fail.Blockchain,
			"network":    fail.Network,
		},
	}
}

func InvalidAccount(fail failure.InvalidAccount) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidAccount,
		Description:     fail.Description,
		Details: map[string]interface{}{
			"address": fail.Address,
			"chain":   fail.Chain,
		},
	}
}

func InvalidCurrency(fail failure.InvalidCurrency) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidCurrency,
		Description:     fail.Description,
		Details: map[string]interface{}{
			"symbol":   fail.Symbol,
			"decimals": fail.Decimals,
		},
	}
}

func InvalidBlock(fail failure.InvalidBlock) Error {
	return RosettaError(
		configuration.ErrorInvalidBlock,
		fail.Details,
		WithString(fail.Hash),
		WithInt(fail.Index),
	)
}

func InvalidTransaction(fail failure.InvalidTransaction) Error {
	return Error{
		ErrorDefinition: configuration.ErrorInvalidTransaction,
		Description:     fail.Description,
		Details: map[string]interface{}{
			"hash": fail.Hash,
		},
	}
}

func UnknownCurrency(fail failure.UnknownCurrency) Error {
	return Error{
		ErrorDefinition: configuration.ErrorUnknownCurrency,
		Description:     fail.Description,
		Details: map[string]interface{}{
			"symbol":   fail.Symbol,
			"decimals": fail.Decimals,
		},
	}
}

func UnknownBlock(fail failure.UnknownBlock) Error {
	return Error{
		ErrorDefinition: configuration.ErrorUnknownBlock,
		Description:     fail.Description,
		Details: map[string]interface{}{
			"index": fail.Index,
			"hash":  fail.Hash,
		},
	}
}

func UnknownTransaction(fail failure.UnknownTransaction) Error {
	return Error{
		ErrorDefinition: configuration.ErrorUnknownTransaction,
		Description:     fail.Description,
		Details: map[string]interface{}{
			"index": fail.Index,
			"hash":  fail.Hash,
		},
	}
}
