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
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"

	"github.com/optakt/flow-dps/models/dps"
)

type Index struct {
	log zerolog.Logger
	lib dps.ReadLibrary
	db  *badger.DB
}

func FromIndex(log zerolog.Logger, lib dps.ReadLibrary, db *badger.DB) *Index {

	i := Index{
		log: log.With().Str("component", "index_loader").Logger(),
		lib: lib,
		db:  db,
	}

	return &i
}

// Trie restores the execution state trie from the DPS index database.
func (i *Index) Trie() (*trie.MTrie, error) {

	processed := 0
	tree := trie.NewEmptyMTrie()
	callback := func(path ledger.Path, payload *ledger.Payload) error {
		var err error
		tree, err = trie.NewTrieWithUpdatedRegisters(tree, []ledger.Path{path}, []ledger.Payload{*payload})
		if err != nil {
			return fmt.Errorf("could not update trie: %w", err)
		}
		processed++
		if processed%10000 == 0 {
			i.log.Debug().Int("processed", processed).Msg("processing registers for trie restoration")
		}
		return nil
	}

	err := i.db.View(i.lib.IterateLedger(callback))
	if err != nil {
		return nil, fmt.Errorf("could not iterate ledger: %w", err)
	}

	return tree, nil
}
