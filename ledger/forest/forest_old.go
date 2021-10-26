// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package forest

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/ledger/forest/trie"
)

type step struct {
	tree   *trie.MTrie
	parent flow.StateCommitment
}

// Forest is a representation of multiple tries mapped by their state commitment hash.
type OtherForest struct {
	values map[ledger.Path]*ledger.Payload
	steps map[flow.StateCommitment]step
}

// New returns a new empty forest.
func New() *OtherForest {
	f := OtherForest{
		steps: make(map[flow.StateCommitment]step),
		values: make(map[ledger.Path]*ledger.Payload),
	}
	return &f
}

// Save adds a tree to the forest.
func (f *OtherForest) Save(tree *trie.MTrie, paths []ledger.Path, payloads []*ledger.Payload, parent flow.StateCommitment) {
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

// Has returns whether a state commitment matches one of the trees within the forest.
func (f *OtherForest) Has(commit flow.StateCommitment) bool {
	_, ok := f.steps[commit]
	return ok
}

// Tree returns the matching tree for the given state commitment.
func (f *OtherForest) Tree(commit flow.StateCommitment) (*trie.MTrie, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return nil, false
	}
	return s.tree, true
}

// Values returns the matching tree's paths for the given state commitment.
func (f *OtherForest) Values() map[ledger.Path]*ledger.Payload {
	return f.values
}

// Parent returns the parent of the given state commitment.
func (f *OtherForest) Parent(commit flow.StateCommitment) (flow.StateCommitment, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return flow.DummyStateCommitment, false
	}
	return s.parent, true
}

// Reset deletes all tries that do not match the given state commitment.
func (f *OtherForest) Reset(finalized flow.StateCommitment) {
	for commit := range f.steps {
		if commit != finalized {
			delete(f.steps, commit)
		}
	}

	// Reset values.
	f.values = make(map[ledger.Path]*ledger.Payload)
}
