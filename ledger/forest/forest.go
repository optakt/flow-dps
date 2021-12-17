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

	"github.com/optakt/flow-dps/ledger/trie"
)

type step struct {
	tree   *trie.Trie
	paths  []ledger.Path
	parent flow.StateCommitment
}

// Forest is a representation of multiple tries mapped by their state commitment hash.
// NOTE: Contrary to the Flow Forest implementation, the forest is unlimited and never evicts any tries.
type Forest struct {
	steps map[flow.StateCommitment]step
}

// New returns a new empty forest.
func New() *Forest {
	f := Forest{
		steps: make(map[flow.StateCommitment]step),
	}

	return &f
}

// Add adds a tree to the forest.
func (f *Forest) Add(tree *trie.Trie, paths []ledger.Path, parent flow.StateCommitment) {
	commit := flow.StateCommitment(tree.RootHash())

	s := step{
		tree:   tree,
		paths:  paths,
		parent: parent,
	}
	f.steps[commit] = s
}

// Has returns whether a state commitment matches one of the trees within the forest.
func (f *Forest) Has(commit flow.StateCommitment) bool {
	_, ok := f.steps[commit]
	return ok
}

// Tree returns the matching tree for the given state commitment.
func (f *Forest) Tree(commit flow.StateCommitment) (*trie.Trie, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return nil, false
	}

	return s.tree, true
}

// Paths returns the matching tree's paths for the given state commitment.
func (f *Forest) Paths(commit flow.StateCommitment) ([]ledger.Path, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return nil, false
	}

	return s.paths, true
}

// Parent returns the parent of the given state commitment.
func (f *Forest) Parent(commit flow.StateCommitment) (flow.StateCommitment, bool) {
	st, ok := f.steps[commit]
	if !ok {
		return flow.DummyStateCommitment, false
	}

	return st.parent, true
}

// Reset deletes all tries that do not match the given state commitment.
func (f *Forest) Reset(finalized flow.StateCommitment) {
	for commit := range f.steps {
		if commit != finalized {
			delete(f.steps, commit)
		}
	}
}

// Trees returns each of the tries from the forest.
func (f *Forest) Trees() []*trie.Trie {
	var tries []*trie.Trie
	for _, step := range f.steps {
		tries = append(tries, step.tree)
	}
	return tries
}
