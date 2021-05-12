package data

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
	"github.com/awfm9/flow-dps/rosetta/object"
)

type Transaction struct {
}

func (t *Transaction) Transaction(network identifier.Network, block identifier.Block, transaction identifier.Transaction) (object.Transaction, error) {
	return object.Transaction{}, nil
}
