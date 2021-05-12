package rosetta

import (
	"github.com/awfm9/flow-dps/model/identifier"
	"github.com/awfm9/flow-dps/model/rosetta"
)

type Retriever interface {
	Block(network identifier.Network, block identifier.Block) (rosetta.Block, []identifier.Transaction, error)
	Transaction(network identifier.Network, block identifier.Block, transaction identifier.Transaction) (rosetta.Transaction, error)
	Balances(network identifier.Network, block identifier.Block, account identifier.Account, currencies []identifier.Currency) ([]rosetta.Amount, error)
}
