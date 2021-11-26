package trie

import (
	"io"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
)

// FIXME: Look into arena allocation for node paths to improve both memory usage and performance.
// FIXME: Look into using a sync Pool to reduce allocations at the expense of some performance.

// Node represents a trie node.
type Node interface {
	Height() uint16
	Path() ledger.Path
	Hash() hash.Hash

	// TODO: The following can be removed if the logic to flatten forests is available through this package rather than done externally. Same goes for listing all paths within a trie from the mapper.

	LeftChild() Node
	RightChild() Node

	Dump(io.Writer)
}
