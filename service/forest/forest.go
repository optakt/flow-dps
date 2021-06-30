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
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
)

type step struct {
	tree   *trie.MTrie
	paths  []ledger.Path
	parent flow.StateCommitment
}

type Forest struct {
	steps map[flow.StateCommitment]step
}

func New() *Forest {
	f := Forest{
		steps: make(map[flow.StateCommitment]step),
	}
	return &f
}

func (f *Forest) Save(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment) {
	commit := flow.StateCommitment(tree.RootHash())
	s := step{
		tree:   tree,
		paths:  paths,
		parent: parent,
	}
	f.steps[commit] = s
}

func (f *Forest) Has(commit flow.StateCommitment) bool {
	_, ok := f.steps[commit]
	return ok
}

func (f *Forest) Tree(commit flow.StateCommitment) (*trie.MTrie, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return nil, false
	}
	return s.tree, true
}

func (f *Forest) Paths(commit flow.StateCommitment) ([]ledger.Path, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return nil, false
	}
	return s.paths, true
}

func (f *Forest) Parent(commit flow.StateCommitment) (flow.StateCommitment, bool) {
	s, ok := f.steps[commit]
	if !ok {
		return flow.StateCommitment{}, false
	}
	return s.parent, true
}

func (f *Forest) Reset(finalized flow.StateCommitment) {
	for commit := range f.steps {
		if commit != finalized {
			delete(f.steps, commit)
		}
	}
}

func (f *Forest) Size() uint {
	return uint(len(f.steps))
}
