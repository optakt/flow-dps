package rosetta

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
	"github.com/awfm9/flow-dps/rosetta/object"
)

type Retriever interface {
	Block(network identifier.Network, block identifier.Block) (object.Block, []identifier.Transaction, error)
	Transaction(network identifier.Network, block identifier.Block, transaction identifier.Transaction) (object.Transaction, error)
	Balances(network identifier.Network, block identifier.Block, account identifier.Account, currencies []identifier.Currency) ([]object.Amount, error)
}
