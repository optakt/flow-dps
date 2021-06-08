// Copyright 2021 Alvalor S.A.
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
	"github.com/optakt/flow-dps/rosetta/rosetta"
)

type BalanceRequest struct {
	NetworkID  identifier.Network    `json:"network_identifier"`
	BlockID    identifier.Block      `json:"block_identifier"`
	AccountID  identifier.Account    `json:"account_identifier"`
	Currencies []identifier.Currency `json:"currencies"`
}

type BalanceResponse struct {
	BlockID  identifier.Block `json:"block_identifier"`
	Balances []rosetta.Amount `json:"balances"`
}

// TODO: integration testing of Rosetta balance endpoint
// => https://github.com/optakt/flow-dps/issues/45
func (d *Data) Balance(ctx echo.Context) error {

	var req BalanceRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat(err.Error()))
	}

	if req.NetworkID.Blockchain == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("blockchain identifier blockchain missing"))
	}
	if req.NetworkID.Network == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("blockchain identifier network missing"))
	}

	if req.BlockID.Index == 0 && req.BlockID.Hash == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("block identifier at least one of hash or index"))
	}
	if req.BlockID.Hash != "" && len(req.BlockID.Hash) != hexIDSize {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("block identifier hash wrong length (have: %d, want: %d)", len(req.BlockID.Hash), hexIDSize))
	}

	if req.AccountID.Address == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("account identifier address missing"))
	}
	if len(req.AccountID.Address) != hexAddressSize {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("account identifier address wrong length (have: %d, want: %d)", len(req.AccountID.Address), hexAddressSize))
	}

	if len(req.Currencies) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("currency identifiers empty"))
	}
	for _, currency := range req.Currencies {
		if currency.Symbol == "" {
			return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("currency identifier symbol missing"))
		}
	}

	err = d.config.Check(req.NetworkID)
	var netErr failure.InvalidNetwork
	if errors.As(err, &netErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidNetwork(netErr))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(err))
	}

	balances, err := d.retrieve.Balances(req.BlockID, req.AccountID, req.Currencies)

	var ibErr failure.InvalidBlock
	if errors.As(err, &ibErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidBlock(ibErr))
	}
	var ubErr failure.UnknownBlock
	if errors.As(err, &ubErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, UnknownBlock(ubErr))
	}

	var iaErr failure.InvalidAccount
	if errors.As(err, &iaErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidAccount(iaErr))
	}

	var icErr failure.InvalidCurrency
	if errors.As(err, &icErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidCurrency(icErr))
	}
	var ucErr failure.UnknownCurrency
	if errors.As(err, &ucErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, UnknownCurrency(ucErr))
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(err))
	}

	res := BalanceResponse{
		BlockID:  req.BlockID,
		Balances: balances,
	}

	return ctx.JSON(http.StatusOK, res)
}
