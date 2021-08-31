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

	sdk "github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

// ParseRequest implements the request schema for /construction/parse.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#request-4
type ParseRequest struct {
	NetworkID   identifier.Network `json:"network_identifier"`
	Signed      bool               `json:"signed"`
	Transaction string             `json:"transaction"`
}

// ParseResponse implements the response schema for /construction/parse.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-4
type ParseResponse struct {
	Operations []object.Operation   `json:"operations"`
	SignerIDs  []identifier.Account `json:"account_identifier_signers,omitempty"`
	Metadata   object.Metadata      `json:"metadata,omitempty"`
}

// Parse implements the /construction/parse endpoint of the Rosetta Construction API.
// Parse endpoint parses both signed and unsigned transactions to understand the
// transaction's intent. Endpoint returns the list of operations, any relevant metadata,
// and, in the case of signed transaction, the list of signers.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#constructionparse
func (c *Construction) Parse(ctx echo.Context) error {

	var req ParseRequest
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

	if req.Transaction == "" {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(txBodyEmpty))
	}

	var tx sdk.Transaction
	err = json.Unmarshal([]byte(req.Transaction), &tx)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(txBodyInvalid, withError(err)))
	}

	// If the transaction is signed, make sure that signatures are provided.
	if req.Signed && len(tx.EnvelopeSignatures) == 0 {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidFormat(txNoSignatures))
	}

	// If the transaction is unsigned, verify that no signatures are present.
	if !req.Signed && len(tx.EnvelopeSignatures) > 0 {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidFormat(txExtraSignatures))
	}

	// Parse the transaction and recreate the original list of operations, as well as all signers involved.
	operations, signers, err := c.parser.ParseTransaction(&tx)

	var iaErr failure.InvalidAuthorizers
	if errors.As(err, &iaErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidAuthorizers(iaErr))
	}
	var iacErr failure.InvalidAccount
	if errors.As(err, &iacErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidAccount(iacErr))
	}
	var ipyErr failure.InvalidPayer
	if errors.As(err, &ipyErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidPayer(ipyErr))
	}
	var iprErr failure.InvalidProposer
	if errors.As(err, &iprErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidProposer(iprErr))
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
	var isgErr failure.InvalidSignature
	if errors.As(err, &isgErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidSignature(isgErr))
	}
	var ibErr failure.InvalidBlock
	if errors.As(err, &ibErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidBlock(ibErr))
	}
	var ubErr failure.UnknownBlock
	if errors.As(err, &ubErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, unknownBlock(ubErr))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, internal(txParsing, err))
	}

	metadata := object.Metadata{
		ReferenceBlockID: identifier.Block{
			Hash: tx.ReferenceBlockID.Hex(),
		},
		SequenceNumber: tx.ProposalKey.SequenceNumber,
	}

	res := ParseResponse{
		Operations: operations,
		SignerIDs:  signers,
		Metadata:   metadata,
	}

	return ctx.JSON(http.StatusOK, res)
}
