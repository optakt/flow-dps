package retriever

import (
	"github.com/awfm9/flow-dps/model/identifier"
	"github.com/awfm9/flow-dps/model/rosetta"
)

func (r *Retriever) Transaction(network identifier.Network, block identifier.Block, transaction identifier.Transaction) (rosetta.Transaction, error) {
	return rosetta.Transaction{}, nil
}
