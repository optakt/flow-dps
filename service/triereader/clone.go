package triereader

import (
	"github.com/onflow/flow-go/ledger"
)

func clone(update *ledger.TrieUpdate) *ledger.TrieUpdate {

	var hash ledger.RootHash
	copy(hash[:], update.RootHash[:])

	paths := make([]ledger.Path, 0, len(update.Paths))
	for _, path := range update.Paths {
		var dup ledger.Path
		copy(dup[:], path[:])
		paths = append(paths, dup)
	}

	payloads := make([]*ledger.Payload, 0, len(update.Payloads))
	for _, payload := range update.Payloads {
		dup := payload.DeepCopy()
		payloads = append(payloads, dup)
	}

	dup := ledger.TrieUpdate{
		RootHash: hash,
		Paths:    paths,
		Payloads: payloads,
	}

	return &dup
}
