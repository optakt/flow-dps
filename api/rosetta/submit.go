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
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/rosetta/identifier"
)

type SubmitRequest struct {
	NetworkID         identifier.Network `json:"network_identifier"`
	SignedTransaction string             `json:"signed_transaction"`
}

type SubmitResponse struct {
	TransactionID identifier.Transaction `json:"transaction_identifier"`
}

func (c *Construction) Submit(ctx echo.Context) error {

	var req HashRequest
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

	if req.SignedTransaction == "" {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("transaction text empty"))
	}

	var tx flow.Transaction
	err = json.Unmarshal([]byte(req.SignedTransaction), &tx)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("could not decode transaction: %w", err))
	}

	err = c.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("could not submit transaction: %w", err))
	}

	res := SubmitResponse{
		TransactionID: identifier.Transaction{
			Hash: tx.ID().Hex(),
		},
	}

	return ctx.JSON(http.StatusOK, res)
}
