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

// BalanceRequest implements the request schema for /account/balance.
// See https://www.rosetta-api.org/docs/AccountApi.html#request
type BalanceRequest struct {
	NetworkID  identifier.Network    `json:"network_identifier"`
	BlockID    identifier.Block      `json:"block_identifier"`
	AccountID  identifier.Account    `json:"account_identifier"`
	Currencies []identifier.Currency `json:"currencies"`
}

// BalanceResponse implements the successful response schema for /account/balance.
// See https://www.rosetta-api.org/docs/AccountApi.html#200---ok
type BalanceResponse struct {
	BlockID  identifier.Block `json:"block_identifier"`
	Balances []object.Amount  `json:"balances"`
}

// Balance implements the /account/balance endpoint of the Rosetta Data API.
// See https://www.rosetta-api.org/docs/AccountApi.html#accountbalance
func (d *Data) Balance(ctx echo.Context) error {

	var req BalanceRequest
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

	if req.BlockID.Index == nil && req.BlockID.Hash == "" {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(blockEmpty))
	}
	if req.BlockID.Hash != "" && len(req.BlockID.Hash) != hexIDSize {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(blockLength,
			withDetail("have_length", len(req.BlockID.Hash)),
			withDetail("want_length", hexIDSize),
		))
	}

	if req.AccountID.Address == "" {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(addressEmpty))
	}
	if len(req.AccountID.Address) != hexAddressSize {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(addressLength,
			withDetail("have_length", len(req.AccountID.Address)),
			withDetail("want_length", hexAddressSize),
		))
	}

	if len(req.Currencies) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(currenciesEmpty))
	}
	for _, currency := range req.Currencies {
		if currency.Symbol == "" {
			return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(symbolEmpty))
		}
	}

	// TODO: Check if we can set up validation middleware to remove the
	// redundant business logic between routes:
	// => https://github.com/optakt/flow-dps/issues/164

	err = d.config.Check(req.NetworkID)
	var netErr failure.InvalidNetwork
	if errors.As(err, &netErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidNetwork(netErr))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, internal(networkCheck, err))
	}

	rosBlockID, balances, err := d.retrieve.Balances(req.BlockID, req.AccountID, req.Currencies)

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

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, internal(balancesRetrieval, err))
	}

	res := BalanceResponse{
		BlockID:  rosBlockID,
		Balances: balances,
	}

	return ctx.JSON(http.StatusOK, res)
}
