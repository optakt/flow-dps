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
	"encoding/json"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

// PayloadsRequest implements the request schema for /construction/payloads.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#request-5
type PayloadsRequest struct {
	NetworkID  identifier.Network `json:"network_identifier"`
	Operations []object.Operation `json:"operations"`
	Metadata   object.Metadata    `json:"metadata"`
}

// PayloadsResponse implements the response schema for /construction/payloads.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-5
type PayloadsResponse struct {
	Transaction string                  `json:"unsigned_transaction"`
	Payloads    []object.SigningPayload `json:"payloads"`
}

// Payloads implements the /construction/payloads endpoint of the Rosetta Construction API.
// It receives an array of operations and all other relevant information required to construct
// an unsigned transaction. Operations must deterministically describe the intent of the
// transaction. Besides the unsigned transaction text, this endpoint also returns the list
// of payloads that should be signed.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#constructionpayloads
func (c *Construction) Payloads(ctx echo.Context) error {

	var req PayloadsRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, invalidEncoding(invalidJSON, err))
	}

	if req.NetworkID.Blockchain == "" {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(blockchainEmpty))
	}
	if req.NetworkID.Network == "" {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(networkEmpty))
	}

	// Metadata object is the response from our metadata endpoint. Thus, the object
	// should be okay, but let's validate it anyway.
	blockID := req.Metadata.ReferenceBlockID
	if blockID.Index == nil && blockID.Hash == "" {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(blockEmpty))
	}
	if blockID.Hash != "" && len(blockID.Hash) != hexIDSize {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(blockLength,
			withDetail("have_length", len(blockID.Hash)),
			withDetail("want_length", hexIDSize),
		))
	}

	intent, err := c.parser.DeriveIntent(req.Operations)
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
	var opErr failure.InvalidOperations
	if errors.As(err, &opErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidFormat(txInvalidOps))
	}
	var inErr failure.InvalidIntent
	if errors.As(err, &inErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidIntent(inErr))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, internal(intentDetermination, err))
	}

	tx, err := c.parser.CompileTransaction(intent, req.Metadata)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, internal(txConstruction, err))
	}

	enc, err := json.Marshal(tx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, internal(txConstruction, err))
	}

	// We only support a single signer at the moment, so the account only needs to sign the transaction envelope.
	res := PayloadsResponse{
		Transaction: string(enc),
		Payloads: []object.SigningPayload{
			{
				AccountID:     identifier.Account{Address: intent.From.Hex()},
				HexBytes:      string(tx.EnvelopeMessage()),
				SignatureType: FlowSignatureAlgorithm,
			},
		},
	}

	return ctx.JSON(http.StatusOK, res)
}
