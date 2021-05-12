package converter

import (
	"strconv"

	"github.com/awfm9/flow-dps/models/identifier"
	"github.com/awfm9/flow-dps/models/rosetta"
)

type Converter struct {
}

func New() *Converter {

	c := &Converter{}

	return c
}

func (c *Converter) Balance(currency identifier.Currency, balance uint64) rosetta.Amount {
	amount := rosetta.Amount{
		Value:    strconv.FormatUint(balance, 10),
		Currency: currency,
	}
	return amount
}
