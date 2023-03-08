package ledgertmp

import (
	"hash"

	"github.com/onflow/flow-go/ledger"
	"github.com/rs/zerolog"
)

// TODO: Move this type def into flow-go/ledger
type LeafNode struct {
	Hash    hash.Hash
	Path    ledger.Path
	Payload *ledger.Payload
}

// TODO: Move this type def into flow-go/ledger
type ReadingResult struct {
	LeafNode *LeafNode
	Err      error
}

// TODO: Move this into flow-go/ledger
func ReadLeafNodeFromCheckpoint(dir string, fileName string, logger *zerolog.Logger) (<-chan *ReadingResult, error) {
	// if checkpoint file not exist, return error
	// otherwise, read the leaf nodes from the given checkpoint
	panic("to be implemented")
}
