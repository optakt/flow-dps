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
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/meta"
)

const (
	invalidJSON = "request does not contain valid JSON-encoded body"

	BlockchainEmpty = "blockchain identifier has empty blockchain field"
	NetworkEmpty    = "blockchain identifier has empty network field"
	BlockNotFull    = "block identifier needs both fields filled for this request"
	BlockLength     = "block identifier has invalid hash field length"
	AddressEmpty    = "account identifier has empty address field"
	AddressLength   = "account identifier has invalid address field length"
	CurrenciesEmpty = "currency identifier list is empty"
	SymbolEmpty     = "currency identifier has empty symbol field"
	TxHashEmpty     = "transaction identifier has empty hash field"
	TxLength        = "transaction identifier has invalid hash field length"
	TxInvalidOps    = "transaction operations are invalid"
	TxBodyEmpty     = "transaction text is empty"
	SignaturesEmpty = "signature list is empty"

	NetworkCheck            = "unable to check network"
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
		withDetail("network", fail.HaveNetwork),
		withDetail("available_networks", fail.AvailableNetworks),
	)
}

func invalidBlockchain(fail failure.InvalidBlockchain) Error {
	return convertError(
		configuration.ErrorInvalidNetwork,
		fail.Description,
		withDetail("blockchain", fail.HaveBlockchain),
		withDetail("available_blockchains", fail.AvailableBlockchains),
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

func invalidPayload(fail failure.InvalidPayload) Error {
	return convertError(
		configuration.ErrorInvalidPayload,
		fail.Description,
		withDetail("encoding", fail.Encoding),
	)
}

// unpackError returns the HTTP status code and Rosetta Error for malformed JSON requests.
func unpackError(err error) *echo.HTTPError {
	return echo.NewHTTPError(http.StatusBadRequest, invalidEncoding(invalidJSON, err))
}

// formatError returns the HTTP status code and Rosetta Error for requests
// that did not pass validation.
func formatError(err error) *echo.HTTPError {

	var ibErr failure.InvalidBlockHash
	if errors.As(err, &ibErr) {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(ibErr.Description.Text,
			withDetail("want_length", ibErr.WantLength),
		))
	}
	var iaErr failure.InvalidAccountAddress
	if errors.As(err, &iaErr) {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(iaErr.Description.Text,
			withDetail("want_length", iaErr.WantLength),
		))
	}
	var itErr failure.InvalidTransactionHash
	if errors.As(err, &itErr) {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(itErr.Description.Text,
			withDetail("want_length", itErr.WantLength),
		))
	}
	var icErr failure.IncompleteBlock
	if errors.As(err, &icErr) {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(icErr.Description.Text))
	}
	var inErr failure.InvalidNetwork
	if errors.As(err, &inErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidNetwork(inErr))
	}
	var iblErr failure.InvalidBlockchain
	if errors.As(err, &iblErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidBlockchain(iblErr))
	}

	return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(err.Error()))
}

// apiError returns the HTTP status code and Rosetta Error for various errors
// occurred during request processing.
func apiError(description string, err error) *echo.HTTPError {

	// Common errors, found both in Data and Construction API.
	var inErr failure.InvalidNetwork
	if errors.As(err, &inErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidNetwork(inErr))
	}
	var ibErr failure.InvalidBlock
	if errors.As(err, &ibErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidBlock(ibErr))
	}
	var ubErr failure.UnknownBlock
	if errors.As(err, &ubErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, unknownBlock(ubErr))
	}
	var iaErr failure.InvalidAccount
	if errors.As(err, &iaErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidAccount(iaErr))
	}
	var icErr failure.InvalidCurrency
	if errors.As(err, &icErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidCurrency(icErr))
	}
	var ucErr failure.UnknownCurrency
	if errors.As(err, &ucErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, unknownCurrency(ucErr))
	}
	var itErr failure.InvalidTransaction
	if errors.As(err, &itErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidTransaction(itErr))
	}
	var utErr failure.UnknownTransaction
	if errors.As(err, &utErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, unknownTransaction(utErr))
	}

	// Construction API specific errors.
	var iautErr failure.InvalidAuthorizers
	if errors.As(err, &iaErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidAuthorizers(iautErr))
	}
	var ipyErr failure.InvalidPayer
	if errors.As(err, &ipyErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidPayer(ipyErr))
	}
	var iprErr failure.InvalidProposer
	if errors.As(err, &iprErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidProposer(iprErr))
	}
	var isgErr failure.InvalidSignature
	if errors.As(err, &isgErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidSignature(isgErr))
	}
	var isgsErr failure.InvalidSignatures
	if errors.As(err, &isgsErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidSignatures(isgsErr))
	}
	var opErr failure.InvalidOperations
	if errors.As(err, &opErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidFormat(TxInvalidOps))
	}
	var intErr failure.InvalidIntent
	if errors.As(err, &intErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidIntent(intErr))
	}
	var ikErr failure.InvalidKey
	if errors.As(err, &ipyErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidKey(ikErr))
	}
	var isErr failure.InvalidScript
	if errors.As(err, &isErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidScript(isErr))
	}
	var iargErr failure.InvalidArguments
	if errors.As(err, &iargErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidArguments(iargErr))
	}
	var imErr failure.InvalidAmount
	if errors.As(err, &imErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidAmount(imErr))
	}
	var irErr failure.InvalidReceiver
	if errors.As(err, &irErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidReceiver(irErr))
	}
	var iplErr failure.InvalidPayload
	if errors.As(err, &iplErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidPayload(iplErr))
	}

	return echo.NewHTTPError(http.StatusInternalServerError, internal(description, err))
}
