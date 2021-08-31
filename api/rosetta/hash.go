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
	"net/http"

	"github.com/labstack/echo/v4"

	sdk "github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/rosetta/identifier"
)

// HashRequest implements the request schema for /construction/hash.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#request-2
type HashRequest struct {
	NetworkID         identifier.Network `json:"network_identifier"`
	SignedTransaction string             `json:"signed_transaction"`
}

// HashResponse implements the response schema for /construction/hash.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-2
type HashResponse struct {
	TransactionID identifier.Transaction `json:"transaction_identifier"`
}

// Hash implements the /construction/hash endpoint of the Rosetta Construction API.
// It returns the transaction ID of a signed transaction.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#constructionhash
func (c *Construction) Hash(ctx echo.Context) error {

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
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(txBodyEmpty))
	}

	var signedTx sdk.Transaction
	err = json.Unmarshal([]byte(req.SignedTransaction), &signedTx)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(txBodyInvalid, withError(err)))
	}

	res := HashResponse{
		TransactionID: identifier.Transaction{
			Hash: signedTx.ID().Hex(),
		},
	}

	return ctx.JSON(http.StatusOK, res)
}
