package loader

import (
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
)

// Empty is a loader that loads as empty execution state trie.
type Empty struct {
}

// FromScratch creates a new loader which loads an empty execution state trie.
func FromScratch() *Empty {

	e := Empty{}

	return &e
}

// Trie returns a freshly initialized empty execution state trie.
func (e *Empty) Trie() (*trie.MTrie, error) {

	tree := trie.NewEmptyMTrie()

	return tree, nil
}
