package retriever

import (
	"github.com/awfm9/flow-dps/model/identifier"
	"github.com/awfm9/flow-dps/model/rosetta"
)

func (r *Retriever) Block(network identifier.Network, block identifier.Block) (rosetta.Block, error) {
	return rosetta.Block{}, nil
}
