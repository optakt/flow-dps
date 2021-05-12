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
)

type Data struct {
	retrieve Retriever
	validate Validator
}

// TODO: implement the error types to return along with the HTTP codes

// TODO: distinguish not found from other errors

func (d *Data) Block(ctx echo.Context) error {

	var req BlockRequest
	err := ctx.Bind(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, nil)
	}

	err = d.validate.Network(req.NetworkID)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, nil)
	}
	err = d.validate.Block(req.BlockID)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, nil)
	}

	block, transactions, err := d.retrieve.Block(req.NetworkID, req.BlockID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, nil)
	}

	res := BlockResponse{
		Block:             block,
		OtherTransactions: transactions,
	}

	return ctx.JSON(http.StatusOK, res)
}

func (d *Data) Transaction(ctx echo.Context) error {

	var req TransactionRequest
	err := ctx.Bind(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, nil)
	}

	err = d.validate.Network(req.NetworkID)
	if err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, nil)
	}
	err = d.validate.Block(req.BlockID)
	if err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, nil)
	}

	transaction, err := d.retrieve.Transaction(req.NetworkID, req.BlockID, req.TransactionID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, nil)
	}

	res := TransactionResponse{
		Transaction: transaction,
	}

	return ctx.JSON(http.StatusOK, res)
}

func (d *Data) Balance(ctx echo.Context) error {

	var req BalanceRequest
	err := ctx.Bind(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, nil)
	}

	err = d.validate.Network(req.NetworkID)
	if err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, nil)
	}
	err = d.validate.Block(req.BlockID)
	if err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, nil)
	}
	for _, currency := range req.Currencies {
		err = d.validate.Currency(currency)
		if err != nil {
			return ctx.JSON(http.StatusUnprocessableEntity, nil)
		}
	}

	balances, err := d.retrieve.Balances(req.NetworkID, req.BlockID, req.AccountID, req.Currencies)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, nil)
	}

	res := BalanceResponse{
		BlockID:  req.BlockID,
		Balances: balances,
	}

	return ctx.JSON(http.StatusOK, res)
}
