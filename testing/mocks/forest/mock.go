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
	"testing"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/testing/mocks"
)

type Mock struct {
	AddFunc    func(tree *trie.Trie, paths []ledger.Path, parent flow.StateCommitment)
	HasFunc    func(commit flow.StateCommitment) bool
	TreeFunc   func(commit flow.StateCommitment) (*trie.Trie, bool)
	PathsFunc  func(commit flow.StateCommitment) ([]ledger.Path, bool)
	ParentFunc func(commit flow.StateCommitment) (flow.StateCommitment, bool)
	ResetFunc  func(finalized flow.StateCommitment)
	SizeFunc   func() uint
}

func BaselineMock(t *testing.T, hasCommit bool) *Mock {
	t.Helper()

	f := Mock{
		AddFunc: func(tree *trie.Trie, paths []ledger.Path, parent flow.StateCommitment) {},
		HasFunc: func(commit flow.StateCommitment) bool {
			return hasCommit
		},
		TreeFunc: func(commit flow.StateCommitment) (*trie.Trie, bool) {
			return trie.NewEmptyTrie(), true
		},
		PathsFunc: func(commit flow.StateCommitment) ([]ledger.Path, bool) {
			return mocks.GenericLedgerPaths(6), true
		},
		ParentFunc: func(commit flow.StateCommitment) (flow.StateCommitment, bool) {
			return mocks.GenericCommit(1), true
		},
		ResetFunc: func(finalized flow.StateCommitment) {},
		SizeFunc: func() uint {
			return 42
		},
	}

	return &f
}

func (f *Mock) Add(tree *trie.Trie, paths []ledger.Path, parent flow.StateCommitment) {
	f.AddFunc(tree, paths, parent)
}

func (f *Mock) Has(commit flow.StateCommitment) bool {
	return f.HasFunc(commit)
}

func (f *Mock) Tree(commit flow.StateCommitment) (*trie.Trie, bool) {
	return f.TreeFunc(commit)
}

func (f *Mock) Paths(commit flow.StateCommitment) ([]ledger.Path, bool) {
	return f.PathsFunc(commit)
}

func (f *Mock) Parent(commit flow.StateCommitment) (flow.StateCommitment, bool) {
	return f.ParentFunc(commit)
}

func (f *Mock) Reset(finalized flow.StateCommitment) {
	f.ResetFunc(finalized)
}

func (f *Mock) Size() uint {
	return f.SizeFunc()
}
