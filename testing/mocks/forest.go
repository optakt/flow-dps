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

package mocks

import (
	"testing"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
)

type Forest struct {
	SaveFunc   func(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment)
	HasFunc    func(commit flow.StateCommitment) bool
	TreeFunc   func(commit flow.StateCommitment) (*trie.MTrie, bool)
	PathsFunc  func(commit flow.StateCommitment) ([]ledger.Path, bool)
	ParentFunc func(commit flow.StateCommitment) (flow.StateCommitment, bool)
	ResetFunc  func(finalized flow.StateCommitment)
	SizeFunc   func() uint
}

func BaselineForest(t *testing.T, hasCommit bool) *Forest {
	t.Helper()

	f := Forest{
		SaveFunc: func(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment) {},
		HasFunc: func(commit flow.StateCommitment) bool {
			return hasCommit
		},
		TreeFunc: func(commit flow.StateCommitment) (*trie.MTrie, bool) {
			return GenericTrie, true
		},
		PathsFunc: func(commit flow.StateCommitment) ([]ledger.Path, bool) {
			return GenericLedgerPaths(6), true
		},
		ParentFunc: func(commit flow.StateCommitment) (flow.StateCommitment, bool) {
			return GenericCommit(1), true
		},
		ResetFunc: func(finalized flow.StateCommitment) {},
		SizeFunc: func() uint {
			return 42
		},
	}

	return &f
}

func (f *Forest) Save(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment) {
	f.SaveFunc(tree, paths, parent)
}

func (f *Forest) Has(commit flow.StateCommitment) bool {
	return f.HasFunc(commit)
}

func (f *Forest) Tree(commit flow.StateCommitment) (*trie.MTrie, bool) {
	return f.TreeFunc(commit)
}

func (f *Forest) Paths(commit flow.StateCommitment) ([]ledger.Path, bool) {
	return f.PathsFunc(commit)
}

func (f *Forest) Parent(commit flow.StateCommitment) (flow.StateCommitment, bool) {
	return f.ParentFunc(commit)
}

func (f *Forest) Reset(finalized flow.StateCommitment) {
	f.ResetFunc(finalized)
}

func (f *Forest) Size() uint {
	return f.SizeFunc()
}
