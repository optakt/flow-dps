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

	"github.com/dgraph-io/badger/v2"

	"github.com/optakt/flow-dps/ledger/trie"
)

type Loader struct {
	TrieFunc func(db *badger.DB) (*trie.Trie, error)
}

func BaselineLoader(t *testing.T) *Loader {
	t.Helper()

	l := Loader{
		TrieFunc: func(*badger.DB) (*trie.Trie, error) {
			return GenericTrie, nil
		},
	}

	return &l
}

func (l *Loader) Trie(db *badger.DB) (*trie.Trie, error) {
	return l.TrieFunc(db)
}
