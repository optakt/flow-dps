package mapper

import (
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
)

// Loader represents something that loads its checkpoint and builds it into a trie.
type Loader interface {
	Trie() (*trie.MTrie, error)
}
