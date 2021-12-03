package forest

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/ledger/trie"
)

type step struct {
	tree   *trie.Trie
	parent flow.StateCommitment
}

// TODO: Look into replacing the forest with a Forest where possible, and improve the Forest
//  to be more performant overall. It could also make sense however to keep this forest implementation
//  for our case and keep another one with more features around.

// Forest is a forest of tries with an unlimited size, but which can be reset manually to remove all tries besides one.
type Forest struct {
	values map[ledger.Path]*ledger.Payload
	steps  map[flow.StateCommitment]step
}

func New() *Forest {
	f := Forest{
		steps:  make(map[flow.StateCommitment]step),
		values: make(map[ledger.Path]*ledger.Payload),
	}

	return &f
}

func (f *Forest) Add(tree *trie.Trie, paths []ledger.Path, payloads []*ledger.Payload, parent flow.StateCommitment) {
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

func (f *Forest) Has(commit flow.StateCommitment) bool {
	_, ok := f.steps[commit]
	return ok
}

func (f *Forest) Tree(commit flow.StateCommitment) (*trie.Trie, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return nil, false
	}

	return s.tree, true
}

func (f *Forest) Parent(commit flow.StateCommitment) (flow.StateCommitment, bool) {
	st, ok := f.steps[commit]
	if !ok {
		return flow.DummyStateCommitment, false
	}

	return st.parent, true
}

func (f *Forest) Values() map[ledger.Path]*ledger.Payload {
	return f.values
}

func (f *Forest) Reset(finalized flow.StateCommitment) {
	for commit := range f.steps {
		if commit != finalized {
			delete(f.steps, commit)
		}
	}
}

// FIXME: The following is only needed for this struct to be used as a flow forest.

func (f *Forest) Read(r *ledger.TrieRead) ([]*ledger.Payload, error) {

	if len(r.Paths) == 0 {
		return []*ledger.Payload{}, nil
	}

	commit, err := flow.ToStateCommitment(r.RootHash[:])
	if err != nil {
		return nil, fmt.Errorf("invalid trie root hash: %w", err)
	}

	trie, ok := f.Tree(commit)
	if !ok {
		return nil, fmt.Errorf("trie does not exist (root hash: %x)", commit[:])
	}

	// Deduplicate keys.
	deduplicatedPaths := make([]ledger.Path, 0, len(r.Paths))
	pathOrgIndex := make(map[ledger.Path][]int)
	for i, path := range r.Paths {
		// only collect duplicated keys once
		indices, ok := pathOrgIndex[path]
		if !ok { // deduplication here is optional
			deduplicatedPaths = append(deduplicatedPaths, path)
		}
		// append the index
		pathOrgIndex[path] = append(indices, i)
	}

	payloads := trie.UnsafeRead(r.Paths)

	// Reconstruct the payloads in the same key order as the given paths.
	orderedPayloads := make([]*ledger.Payload, len(r.Paths))
	totalPayloadSize := 0
	for i, p := range deduplicatedPaths {
		payload := payloads[i]
		indices := pathOrgIndex[p]
		for _, j := range indices {
			orderedPayloads[j] = payload.DeepCopy()
		}
		totalPayloadSize += len(indices) * payload.Size()
	}

	return orderedPayloads, nil
}

// FIXME: Is it an issue that instead of duplicating the trie and adding the update to it, we simply update the existing trie?

// Update applies a trie update to the trie which matches its root hash, within the forest.
func (f *Forest) Update(u *ledger.TrieUpdate) (ledger.RootHash, error) {
	if len(u.Paths) == 0 {
		return u.RootHash, nil
	}

	commit, err := flow.ToStateCommitment(u.RootHash[:])
	if err != nil {
		return ledger.RootHash(hash.DummyHash), fmt.Errorf("invalid trie root hash: %w", err)
	}

	tree, ok := f.Tree(commit)
	if !ok {
		return ledger.RootHash(hash.DummyHash), fmt.Errorf("trie does not exist (root hash: %x)", commit[:])
	}

	// Deduplicate paths.
	deduplicatedPaths := make([]ledger.Path, 0, len(u.Paths))
	deduplicatedPayloads := make([]ledger.Payload, 0, len(u.Paths))
	payloadMap := make(map[ledger.Path]int) // index into deduplicatedPaths, deduplicatedPayloads with register update
	for i, path := range u.Paths {
		payload := u.Payloads[i]
		// check if we already have encountered an update for the respective register
		if idx, ok := payloadMap[path]; ok {
			deduplicatedPayloads[idx] = *payload
		} else {
			payloadMap[path] = len(deduplicatedPaths)
			deduplicatedPaths = append(deduplicatedPaths, path)
			deduplicatedPayloads = append(deduplicatedPayloads, *u.Payloads[i])
		}
	}

	for i := range deduplicatedPaths {
		tree.Insert(deduplicatedPaths[i], &deduplicatedPayloads[i])
	}

	return tree.RootHash(), nil
}

func (f *Forest) GetTries() ([]*trie.Trie, error) {
	var tries []*trie.Trie
	for _, step := range f.steps {
		tries = append(tries, step.tree)
	}
	return tries, nil
}

func (f *Forest) AddTries(newTries []*trie.Trie) error {
	for _, t := range newTries {
		f.Add(t, nil, nil, flow.DummyStateCommitment)
	}

	return nil
}
