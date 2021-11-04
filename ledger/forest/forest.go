package forest

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/ledger/trie"
)

type step struct {
	tree trie.Trie
}

type Forest struct {
	values map[ledger.Path]*ledger.Payload
	steps map[flow.StateCommitment]step
}

func New() *Forest {
	f := Forest{
		steps: make(map[flow.StateCommitment]step),
		values: make(map[ledger.Path]*ledger.Payload),
	}

	return &f
}


func (f *Forest) Save(tree *trie.LightTrie, paths []ledger.Path, payloads []*ledger.Payload, parent flow.StateCommitment) {
	commit := flow.StateCommitment(tree.RootHash())

	for i := range paths {
		if payloads == nil {
			f.values[paths[i]] = nil
		} else {
			f.values[paths[i]] = payloads[i]
		}
	}

	s := step{
		tree:   tree,
		parent: parent,
	}
	f.steps[commit] = s
}