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
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type BalanceRequest struct {
	NetworkID  identifier.Network    `json:"network_identifier"`
	BlockID    identifier.Block      `json:"block_identifier"`
	AccountID  identifier.Account    `json:"account_identifier"`
	Currencies []identifier.Currency `json:"currencies"`
}

type BalanceResponse struct {
	BlockID  identifier.Block `json:"block_identifier"`
	Balances []object.Amount  `json:"balances"`
}

// TODO: integration testing of Rosetta balance endpoint
// => https://github.com/optakt/flow-dps/issues/45
func (d *Data) Balance(ctx echo.Context) error {

	var req BalanceRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat(err))
	}

	balances, err := d.retrieve.Balances(req.NetworkID, req.BlockID, req.AccountID, req.Currencies)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(err))
	}

	res := BalanceResponse{
		BlockID:  req.BlockID,
		Balances: balances,
	}

	return ctx.JSON(http.StatusOK, res)
}
