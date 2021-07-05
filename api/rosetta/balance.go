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

	"github.com/optakt/flow-dps/rosetta/fail"
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

func (d *Data) Balance(ctx echo.Context) error {

	var req BalanceRequest
	err := ctx.Bind(&req)
	if err != nil {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("could not unmarshal request", fail.WithError(err)))
	}

	if req.NetworkID.Blockchain == "" {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("blockchain identifier: blockchain field is empty"))
	}
	if req.NetworkID.Network == "" {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("blockchain identifier: network field is empty"))
	}

	if req.BlockID.Index == 0 && req.BlockID.Hash == "" {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("block identifier: at least one of hash or index is required"))
	}
	if req.BlockID.Hash != "" && len(req.BlockID.Hash) != hexIDSize {
		return httpError(
			http.StatusBadRequest,
			fail.InvalidFormat("block identifier: hash field has wrong length",
				fail.WithInt("have_length", len(req.BlockID.Hash)),
				fail.WithInt("want_length", hexIDSize),
			))
	}

	if req.AccountID.Address == "" {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("account identifier: address field is empty"))
	}
	if len(req.AccountID.Address) != hexAddressSize {

		return httpError(
			http.StatusBadRequest,
			fail.InvalidFormat("account identifier: address field has wrong length",
				fail.WithInt("have_length", len(req.AccountID.Address)),
				fail.WithInt("want_length", hexAddressSize),
			))
	}

	if len(req.Currencies) == 0 {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("currency identifiers: currency list is empty"))
	}
	for _, currency := range req.Currencies {
		if currency.Symbol == "" {
			return httpError(http.StatusBadRequest, fail.InvalidFormat("currency identifier: symbol field is missing"))
		}
	}

	// TODO: Check if we can set up validation middleware to remove the
	// redundant business logic between routes:
	// => https://github.com/optakt/flow-dps/issues/164

	err = d.config.Check(req.NetworkID)
	var netErr fail.InvalidNetwork
	if errors.As(err, &netErr) {
		return httpError(http.StatusUnprocessableEntity, netErr.RosettaError())
	}
	if err != nil {
		return httpError(http.StatusInternalServerError, fail.Internal("could not validate network", fail.WithError(err)))
	}

	block, balances, err := d.retrieve.Balances(req.BlockID, req.AccountID, req.Currencies)

	var ibErr fail.InvalidBlock
	if errors.As(err, &ibErr) {
		return httpError(http.StatusUnprocessableEntity, ibErr.RosettaError())
	}
	var ubErr fail.UnknownBlock
	if errors.As(err, &ubErr) {
		return httpError(http.StatusUnprocessableEntity, ubErr.RosettaError())
	}

	var iaErr fail.InvalidAccount
	if errors.As(err, &iaErr) {
		return httpError(http.StatusUnprocessableEntity, iaErr.RosettaError())
	}

	var icErr fail.InvalidCurrency
	if errors.As(err, &icErr) {
		return httpError(http.StatusUnprocessableEntity, icErr.RosettaError())
	}
	var ucErr fail.UnknownCurrency
	if errors.As(err, &ucErr) {
		return httpError(http.StatusUnprocessableEntity, ucErr.RosettaError())
	}

	if err != nil {
		return httpError(http.StatusInternalServerError, fail.Internal("could not retrieve balance", fail.WithError(err)))
	}

	res := BalanceResponse{
		BlockID:  block,
		Balances: balances,
	}

	return ctx.JSON(http.StatusOK, res)
}
