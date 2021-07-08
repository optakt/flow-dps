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

type DetailFunc func(map[string]interface{})

func WithError(err error) DetailFunc {
	return func(details map[string]interface{}) {
		details["error"] = err.Error()
	}
}

func WithDetail(key string, val interface{}) DetailFunc {
	return func(details map[string]interface{}) {
		details[key] = val
	}
}

func RosettaError(definition meta.ErrorDefinition, description string, details ...DetailFunc) Error {
	dd := make(map[string]interface{})
	for _, detail := range details {
		detail(dd)
	}
	e := Error{
		ErrorDefinition: definition,
		Description:     description,
		Details:         dd,
	}
	return e
}

func Internal(description string, err error) Error {
	return RosettaError(
		configuration.ErrorInternal,
		description,
		WithError(err),
	)
}

func InvalidEncoding(description string, err error) Error {
	return RosettaError(
		configuration.ErrorInvalidEncoding,
		description,
		WithError(err),
	)
}

func InvalidFormat(description string, details ...DetailFunc) Error {
	return RosettaError(
		configuration.ErrorInvalidFormat,
		description,
		details...,
	)
}

func ConvertError(definition meta.ErrorDefinition, description failure.Description, details ...DetailFunc) Error {
	description.Fields.Iterate(func(key string, val interface{}) {
		details = append(details, WithDetail(key, val))
	})
	return RosettaError(definition, description.Text, details...)
}

func InvalidNetwork(fail failure.InvalidNetwork) Error {
	return ConvertError(
		configuration.ErrorInvalidNetwork,
		fail.Description,
		WithDetail("blockchain", fail.Blockchain),
		WithDetail("network", fail.Network),
	)
}

func InvalidAccount(fail failure.InvalidAccount) Error {
	return ConvertError(
		configuration.ErrorInvalidAccount,
		fail.Description,
		WithDetail("address", fail.Address),
	)
}

func InvalidCurrency(fail failure.InvalidCurrency) Error {
	return ConvertError(
		configuration.ErrorInvalidCurrency,
		fail.Description,
		WithDetail("symbol", fail.Symbol),
		WithDetail("decimals", fail.Decimals),
	)
}

func InvalidBlock(fail failure.InvalidBlock) Error {
	return ConvertError(
		configuration.ErrorInvalidBlock,
		fail.Description,
		WithDetail("index", fail.Index),
		WithDetail("hash", fail.Hash),
	)
}

func InvalidTransaction(fail failure.InvalidTransaction) Error {
	return ConvertError(
		configuration.ErrorInvalidTransaction,
		fail.Description,
		WithDetail("hash", fail.Hash),
	)
}

func UnknownCurrency(fail failure.UnknownCurrency) Error {
	return ConvertError(
		configuration.ErrorUnknownCurrency,
		fail.Description,
		WithDetail("symbol", fail.Symbol),
		WithDetail("decimals", fail.Decimals),
	)
}

func UnknownBlock(fail failure.UnknownBlock) Error {
	return ConvertError(
		configuration.ErrorUnknownBlock,
		fail.Description,
		WithDetail("index", fail.Index),
		WithDetail("hash", fail.Hash),
	)
}

func UnknownTransaction(fail failure.UnknownTransaction) Error {
	return ConvertError(
		configuration.ErrorUnknownTransaction,
		fail.Description,
		WithDetail("hash", fail.Hash),
	)
}
