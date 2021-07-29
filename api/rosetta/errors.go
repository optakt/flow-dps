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

const (
	invalidJSON = "request does not contain valid JSON-encoded body"

	blockchainEmpty = "blockchain identifier has empty blockchain field"
	networkEmpty    = "blockchain identifier has empty network field"
	blockEmpty      = "block identifier has empty index and hash fields"
	blockLength     = "block identifier has invalid hash field length"
	addressEmpty    = "account identifier has empty address field"
	addressLength   = "account identifier has invalid address field length"
	currenciesEmpty = "currency identifier list is empty"
	symbolEmpty     = "currency identifier has empty symbol field"
	txEmpty         = "transaction identifier has empty hash field"
	txLength        = "transaction identifier has invalid hash field length"

	networkCheck      = "unable to check network"
	blockRetrieval    = "unable to retrieve block"
	balancesRetrieval = "unable to retrieve balances"
	oldestRetrieval   = "unable to retrieve oldest block"
	currentRetrieval  = "unable to retrieve current block"
	txRetrieval       = "unable to retrieve transaction"
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

type detailFunc func(map[string]interface{})

func withError(err error) detailFunc {
	return func(details map[string]interface{}) {
		details["error"] = err.Error()
	}
}

func withDetail(key string, val interface{}) detailFunc {
	return func(details map[string]interface{}) {
		details[key] = val
	}
}

func rosettaError(definition meta.ErrorDefinition, description string, details ...detailFunc) Error {
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

func internal(description string, err error) Error {
	return rosettaError(
		configuration.ErrorInternal,
		description,
		withError(err),
	)
}

func invalidEncoding(description string, err error) Error {
	return rosettaError(
		configuration.ErrorInvalidEncoding,
		description,
		withError(err),
	)
}

func invalidFormat(description string, details ...detailFunc) Error {
	return rosettaError(
		configuration.ErrorInvalidFormat,
		description,
		details...,
	)
}

func convertError(definition meta.ErrorDefinition, description failure.Description, details ...detailFunc) Error {
	description.Fields.Iterate(func(key string, val interface{}) {
		details = append(details, withDetail(key, val))
	})
	return rosettaError(definition, description.Text, details...)
}

func invalidNetwork(fail failure.InvalidNetwork) Error {
	return convertError(
		configuration.ErrorInvalidNetwork,
		fail.Description,
		withDetail("blockchain", fail.Blockchain),
		withDetail("network", fail.Network),
	)
}

func invalidAccount(fail failure.InvalidAccount) Error {
	return convertError(
		configuration.ErrorInvalidAccount,
		fail.Description,
		withDetail("address", fail.Address),
	)
}

func invalidCurrency(fail failure.InvalidCurrency) Error {
	return convertError(
		configuration.ErrorInvalidCurrency,
		fail.Description,
		withDetail("symbol", fail.Symbol),
		withDetail("decimals", fail.Decimals),
	)
}

func invalidBlock(fail failure.InvalidBlock) Error {
	return convertError(
		configuration.ErrorInvalidBlock,
		fail.Description,
	)
}

func invalidTransaction(fail failure.InvalidTransaction) Error {
	return convertError(
		configuration.ErrorInvalidTransaction,
		fail.Description,
		withDetail("hash", fail.Hash),
	)
}

func unknownCurrency(fail failure.UnknownCurrency) Error {
	return convertError(
		configuration.ErrorUnknownCurrency,
		fail.Description,
		withDetail("symbol", fail.Symbol),
		withDetail("decimals", fail.Decimals),
	)
}

func unknownBlock(fail failure.UnknownBlock) Error {
	return convertError(
		configuration.ErrorUnknownBlock,
		fail.Description,
		withDetail("index", fail.Index),
		withDetail("hash", fail.Hash),
	)
}

func unknownTransaction(fail failure.UnknownTransaction) Error {
	return convertError(
		configuration.ErrorUnknownTransaction,
		fail.Description,
		withDetail("hash", fail.Hash),
	)
}
