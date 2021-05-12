package retriever

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
	"github.com/awfm9/flow-dps/rosetta/object"
)

func (r *Retriever) Block(network identifier.Network, block identifier.Block) (object.Block, error) {
	return object.Block{}, nil
}
