package retriever

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
	"github.com/awfm9/flow-dps/rosetta/object"
)

func (r *Retriever) Transaction(network identifier.Network, block identifier.Block, transaction identifier.Transaction) (object.Transaction, error) {
	return object.Transaction{}, nil
}
