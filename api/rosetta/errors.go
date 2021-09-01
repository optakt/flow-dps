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
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/meta"
)

const (
	invalidJSON = "request does not contain valid JSON-encoded body"

	blockchainEmpty = "blockchain identifier has empty blockchain field"
	networkEmpty    = "blockchain identifier has empty network field"
	blockNotFull    = "block identifier needs both fields filled for this request"
	blockLength     = "block identifier has invalid hash field length"
	addressEmpty    = "account identifier has empty address field"
	addressLength   = "account identifier has invalid address field length"
	currenciesEmpty = "currency identifier list is empty"
	symbolEmpty     = "currency identifier has empty symbol field"
	txHashEmpty     = "transaction identifier has empty hash field"
	txLength        = "transaction identifier has invalid hash filed length"
	txInvalidOps    = "transaction operations are invalid"
	txBodyEmpty     = "transaction text is empty"
	signaturesEmpty = "signature list is empty"

	networkCheck            = "unable to check network"
	blockRetrieval          = "unable to retrieve block"
	balancesRetrieval       = "unable to retrieve balances"
	oldestRetrieval         = "unable to retrieve oldest block"
	currentRetrieval        = "unable to retrieve current block"
	txSubmission            = "unable to submit transaction"
	txRetrieval             = "unable to retrieve transaction"
	intentDetermination     = "unable to determine transaction intent"
	referenceBlockRetrieval = "unable to retrieve transaction reference block"
	sequenceNumberRetrieval = "unable to retrieve account key sequence number"
	txConstruction          = "unable to construct transaction"
	txParsing               = "unable to parse transaction"
	txSigning               = "unable to sign transaction"
	payloadHashing          = "unable to hash signing payload"
	txIdentifier            = "unable to retrieve transaction identifier"
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

func withAddress(key string, val flow.Address) detailFunc {
	return func(details map[string]interface{}) {
		details[key] = val.Hex()
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

func invalidIntent(fail failure.InvalidIntent) Error {
	return convertError(
		configuration.ErrorInvalidIntent,
		fail.Description,
	)
}

func invalidAuthorizers(fail failure.InvalidAuthorizers) Error {
	return convertError(
		configuration.ErrorInvalidAuthorizers,
		fail.Description,
		withDetail("have_authorizers", fail.Have),
		withDetail("want_authorizers", fail.Want),
	)
}

func invalidSignatures(fail failure.InvalidSignatures) Error {
	return convertError(
		configuration.ErrorInvalidSignatures,
		fail.Description,
		withDetail("have_signatures", fail.Have),
		withDetail("want_signatures", fail.Want),
	)
}

func invalidPayer(fail failure.InvalidPayer) Error {
	return convertError(
		configuration.ErrorInvalidPayer,
		fail.Description,
		withAddress("have_payer", fail.Have),
		withAddress("want_payer", fail.Want),
	)
}

func invalidProposer(fail failure.InvalidProposer) Error {
	return convertError(
		configuration.ErrorInvalidProposer,
		fail.Description,
		withAddress("have_proposer", fail.Have),
		withAddress("want_proposer", fail.Want),
	)
}

func invalidScript(fail failure.InvalidScript) Error {
	return convertError(
		configuration.ErrorInvalidScript,
		fail.Description,
		withDetail("script", fail.Script),
	)
}

func invalidArguments(fail failure.InvalidArguments) Error {
	return convertError(
		configuration.ErrorInvalidArguments,
		fail.Description,
		withDetail("have_arguments", fail.Have),
		withDetail("want_arguments", fail.Want),
	)
}

func invalidAmount(fail failure.InvalidAmount) Error {
	return convertError(
		configuration.ErrorInvalidAmount,
		fail.Description,
		withDetail("amount", fail.Amount),
	)
}

func invalidReceiver(fail failure.InvalidReceiver) Error {
	return convertError(
		configuration.ErrorInvalidReceiver,
		fail.Description,
		withDetail("receiver", fail.Receiver),
	)
}

func invalidSignature(fail failure.InvalidSignature) Error {
	return convertError(
		configuration.ErrorInvalidSignature,
		fail.Description,
	)
}

func invalidKey(fail failure.InvalidKey) Error {
	return convertError(
		configuration.ErrorInvalidKey,
		fail.Description,
		withDetail("height", fail.Height),
		withAddress("account", fail.Address),
		withDetail("index", fail.Index),
	)
}
