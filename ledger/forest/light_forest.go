package forest

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/ledger/trie"
)

type step struct {
	tree *trie.Trie
	parent flow.StateCommitment
}

// TODO: Look into replacing the forest with a LightForest where possible, and improve the LightForest
//  to be more performant overall. It could also make sense however to keep this forest implementation
//  for our case and keep another one with more features around.

type LightForest struct {
	values map[ledger.Path]*ledger.Payload
	steps map[flow.StateCommitment]step
}

func New() *LightForest {
	f := LightForest{
		steps: make(map[flow.StateCommitment]step),
		values: make(map[ledger.Path]*ledger.Payload),
	}

	return &f
}

func (f *LightForest) Add(tree *trie.Trie, paths []ledger.Path, payloads []*ledger.Payload, parent flow.StateCommitment) {
	commit := flow.StateCommitment(tree.RootHash())

	// FIXME: Is this still necessary? IIRC it was a temporary solution because we didn't have access to payloads anymore. Since we can list the paths and have access to the DB, we might do that instead of storing values in there.
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

func (f *LightForest) Has(commit flow.StateCommitment) bool {
	_, ok := f.steps[commit]
	return ok
}

func (f *LightForest) Tree(commit flow.StateCommitment) (*trie.Trie, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return nil, false
	}

	return s.tree, true
}

func (f *LightForest) Parent(commit flow.StateCommitment) (flow.StateCommitment, bool) {
	st, ok := f.steps[commit]
	if !ok {
		return flow.DummyStateCommitment, false
	}

	return st.parent, true
}

func (f *LightForest) Values() map[ledger.Path]*ledger.Payload {
	return f.values
}

func (f *LightForest) Reset(finalized flow.StateCommitment) {
	for commit := range f.steps {
		if commit != finalized {
			delete(f.steps, commit)
		}
	}
}
