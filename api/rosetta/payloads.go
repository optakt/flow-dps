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
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type PayloadsRequest struct {
	NetworkID  identifier.Network `json:"network_identifier"`
	Operations []object.Operation `json:"operations"`
	Metadata   object.Metadata    `json:"metadata"`
}

type PayloadsResponse struct {
	Transaction string                  `json:"unsigned_transaction"`
	Payloads    []object.SigningPayload `json:"payloads"`
}

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

	if len(req.Operations) != 2 {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(txInvalidOpCount))
	}

	intent, err := c.parser.CreateTransactionIntent(req.Operations)
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
	var inErr failure.InvalidIntent
	if errors.As(err, &inErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidIntent(inErr))
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, internal(txPreprocess, err))
	}

	tx, err := c.parser.CreateTransaction(intent)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, fmt.Errorf("could not create transaction: %w", err))
	}

	enc, err := json.Marshal(tx)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, fmt.Errorf("could not encode transaction: %w", err))
	}

	res := PayloadsResponse{
		Transaction: string(enc),
		Payloads: []object.SigningPayload{
			{
				AccountID:     identifier.Account{Address: intent.From.Hex()},
				HexBytes:      string(tx.PayloadMessage()),
				SignatureType: FlowSignatureAlgorithm,
			},
		},
	}

	return ctx.JSON(http.StatusOK, res)
}
