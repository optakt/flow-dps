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

package loader

import (
	"github.com/rs/zerolog"

	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/models/dps"
)

// Empty is a loader that loads as empty execution state trie.
type Empty struct {
	log   zerolog.Logger
	store dps.Store
}

// FromScratch creates a new loader which loads an empty execution state trie.
func FromScratch(log zerolog.Logger, store dps.Store) *Empty {

	e := Empty{
		log:   log,
		store: store,
	}

	return &e
}

// Trie returns a freshly initialized empty execution state trie.
func (e *Empty) Trie() (*trie.Trie, error) {

	tree := trie.NewEmptyTrie(e.log, e.store)

	return tree, nil
}
