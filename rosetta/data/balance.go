package data

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
	"github.com/awfm9/flow-dps/rosetta/object"
)

type Balance struct {
}

func (b *Balance) Balance(network identifier.Network, block identifier.Block, account identifier.Account, currencies []identifier.Currency) ([]object.Amount, error) {
	return nil, nil
}
