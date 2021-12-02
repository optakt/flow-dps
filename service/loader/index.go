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
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"

	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/models/dps"
)

// Index implements an execution state trie loader on top of a DPS index,
// able to restore an execution state trie from the index database.
type Index struct {
	log zerolog.Logger
	lib dps.ReadLibrary
	cfg Config

	// db is the database that contains the index.
	db *badger.DB

	// store is the payload storage to use for the trie that is generated from the index.
	store dps.Store
}

// FromIndex creates a new index loader, which can restore the execution state
// from the given index database, using the given library for decoding ledger
// paths and payloads.
func FromIndex(log zerolog.Logger, lib dps.ReadLibrary, db *badger.DB, store dps.Store, options ...Option) *Index {

	cfg := DefaultConfig
	for _, option := range options {
		option(&cfg)
	}

	i := Index{
		log:   log.With().Str("component", "index_loader").Logger(),
		lib:   lib,
		db:    db,
		store: store,
		cfg:   cfg,
	}

	return &i
}

// Trie restores the execution state trie from the DPS index database, as it was
// when indexing was stopped.
func (i *Index) Trie() (*trie.Trie, error) {

	// Load the starting trie.
	tree, err := i.cfg.TrieInitializer.Trie()
	if err != nil {
		return nil, fmt.Errorf("could not initialize trie: %w", err)
	}

	trie := trie.NewEmptyTrie(i.store)
	processed := 0
	process := func(path ledger.Path, payload *ledger.Payload) error {
		trie.Insert(path, payload)
		processed++
		if processed%10000 == 0 {
			i.log.Debug().Int("processed", processed).Msg("processing registers for trie restoration")
		}
		return nil
	}

	err = i.db.View(i.lib.IterateLedger(i.cfg.ExcludeHeight, process))
	if err != nil {
		return nil, fmt.Errorf("could not iterate ledger: %w", err)
	}

	return tree, nil
}
