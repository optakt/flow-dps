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
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/ledger/trie"
)

type Forest struct {
	AddFunc    func(tree *trie.Trie, paths []ledger.Path, payloads []*ledger.Payload, parent flow.StateCommitment)
	HasFunc    func(commit flow.StateCommitment) bool
	TreeFunc   func(commit flow.StateCommitment) (*trie.Trie, bool)
	PathsFunc  func(commit flow.StateCommitment) ([]ledger.Path, bool)
	ParentFunc func(commit flow.StateCommitment) (flow.StateCommitment, bool)
	ValuesFunc func() map[ledger.Path]*ledger.Payload
	ResetFunc  func(finalized flow.StateCommitment)
	SizeFunc   func() uint
}

func BaselineForest(t *testing.T, hasCommit bool) *Forest {
	t.Helper()

	f := Forest{
		AddFunc: func(tree *trie.Trie, paths []ledger.Path, payloads []*ledger.Payload, parent flow.StateCommitment) {},
		HasFunc: func(commit flow.StateCommitment) bool {
			return hasCommit
		},
		TreeFunc: func(commit flow.StateCommitment) (*trie.Trie, bool) {
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

func (f *Forest) Add(tree *trie.Trie, paths []ledger.Path, payloads []*ledger.Payload, parent flow.StateCommitment) {
	f.AddFunc(tree, paths, payloads, parent)
}

func (f *Forest) Has(commit flow.StateCommitment) bool {
	return f.HasFunc(commit)
}

func (f *Forest) Tree(commit flow.StateCommitment) (*trie.Trie, bool) {
	return f.TreeFunc(commit)
}

func (f *Forest) Paths(commit flow.StateCommitment) ([]ledger.Path, bool) {
	return f.PathsFunc(commit)
}

func (f *Forest) Parent(commit flow.StateCommitment) (flow.StateCommitment, bool) {
	return f.ParentFunc(commit)
}

func (f *Forest) Values() map[ledger.Path]*ledger.Payload {
	return f.ValuesFunc()
}

func (f *Forest) Reset(finalized flow.StateCommitment) {
	f.ResetFunc(finalized)
}

func (f *Forest) Size() uint {
	return f.SizeFunc()
}
