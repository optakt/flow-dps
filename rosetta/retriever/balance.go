package retriever

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
	"github.com/awfm9/flow-dps/rosetta/object"
)

func (r *Retriever) Balances(network identifier.Network, block identifier.Block, account identifier.Account, currencies []identifier.Currency) ([]object.Amount, error) {
	return nil, nil
}
