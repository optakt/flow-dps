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

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

// TransactionRequest implements the request schema for /block/transaction.
// See https://www.rosetta-api.org/docs/BlockApi.html#request-1
type TransactionRequest struct {
	NetworkID     identifier.Network     `json:"network_identifier"`
	BlockID       identifier.Block       `json:"block_identifier"`
	TransactionID identifier.Transaction `json:"transaction_identifier"`
}

// TransactionResponse implements the successful response schema for /block/transaction.
// See https://www.rosetta-api.org/docs/BlockApi.html#200---ok-1
type TransactionResponse struct {
	Transaction *object.Transaction `json:"transaction"`
}

// Transaction implements the /block/transaction endpoint of the Rosetta Data API.
// See https://www.rosetta-api.org/docs/BlockApi.html#blocktransaction
func (d *Data) Transaction(ctx echo.Context) error {

	var req TransactionRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidEncoding("request does not contain valid JSON-encoded body", err))
	}

	if req.NetworkID.Blockchain == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("network identifier has empty blockchain field"))
	}
	if req.NetworkID.Network == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("network identifier has empty network field"))
	}

	if req.BlockID.Index == nil && req.BlockID.Hash == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("block identifier has empty index and hash fields"))
	}
	if req.BlockID.Hash != "" && len(req.BlockID.Hash) != hexIDSize {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("block identifier has invalid hash field length",
			WithDetail("have_length", len(req.BlockID.Hash)),
			WithDetail("want_length", hexIDSize),
		))
	}

	if req.TransactionID.Hash == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("transaction identifier has empty hash field"))
	}
	if len(req.TransactionID.Hash) != hexIDSize {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("transaction identifier has invalid hash filed length",
			WithDetail("have_length", len(req.TransactionID.Hash)),
			WithDetail("want_length", hexIDSize),
		))
	}

	err = d.config.Check(req.NetworkID)
	var netErr failure.InvalidNetwork
	if errors.As(err, &netErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidNetwork(netErr))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal("unable to check network configuration", err))
	}

	transaction, err := d.retrieve.Transaction(req.BlockID, req.TransactionID)

	var ibErr failure.InvalidBlock
	if errors.As(err, &ibErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidBlock(ibErr))
	}
	var ubErr failure.UnknownBlock
	if errors.As(err, &ubErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, UnknownBlock(ubErr))
	}

	var itErr failure.InvalidTransaction
	if errors.As(err, &itErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidTransaction(itErr))
	}
	var utErr failure.UnknownTransaction
	if errors.As(err, &utErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, UnknownTransaction(utErr))
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal("unable to retrieve transaction", err))
	}

	res := TransactionResponse{
		Transaction: transaction,
	}

	return ctx.JSON(http.StatusOK, res)
}
