package mapper

import (
	"github.com/onflow/flow-go/ledger"
)

// TrieUpdates represents something to get trie updates in block-height order.
type TrieUpdates interface {
	AllUpdates() ([]*ledger.TrieUpdate, error)
}
