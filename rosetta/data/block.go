package data

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
	"github.com/awfm9/flow-dps/rosetta/object"
)

type Block struct {
}

func (b *Block) Block(network identifier.Network, block identifier.Block) (object.Block, error) {
	return object.Block{}, nil
}
