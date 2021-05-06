package rest

import (
	"github.com/labstack/echo/v4"
)

type Controller struct {
	raw    Raw
	ledger Ledger
}

func NewController(raw Raw, ledger Ledger) (*Controller, error) {
	c := &Controller{
		raw:    raw,
		ledger: ledger,
	}
	return c, nil
}

func (c *Controller) GetRegister(ctx echo.Context) error {
	return nil
}

func (c *Controller) GetPayload(ctx echo.Context) error {
	return nil
}
