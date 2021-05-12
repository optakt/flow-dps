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
