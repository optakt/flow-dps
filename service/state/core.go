// Copyright 2021 Alvalor S.A.
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

package state

import (
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/storage"
)

// TODO: improve code comments & documentation throughout the refactored
// DPS architecture & components
// => https://github.com/optakt/flow-dps/issues/40

type Core struct {
	db     *badger.DB
	height uint64
	commit flow.StateCommitment
}

func NewCore(dir string) (*Core, error) {

	opts := badger.DefaultOptions(dir).
		WithMaxTableSize(256 << 20).
		WithValueLogFileSize(64 << 20).
		WithTableLoadingMode(options.FileIO).
		WithValueLogLoadingMode(options.FileIO).
		WithNumMemtables(1).
		WithKeepL0InMemory(false).
		WithCompactL0OnClose(false).
		WithNumLevelZeroTables(1).
		WithNumLevelZeroTablesStall(2).
		WithLoadBloomsOnOpen(false).
		WithIndexCacheSize(2000 << 20).
		WithBlockCacheSize(0).
		WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	var commit flow.StateCommitment
	err = db.View(storage.RetrieveLastCommit(&commit))
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("could not retrieve last commit: %w", err)
	}
	if errors.Is(err, badger.ErrKeyNotFound) {
		tree := trie.NewEmptyMTrie()
		commit = flow.StateCommitment(tree.RootHash())
		err = db.Update(storage.Combine(
			storage.SaveLastCommit(commit),
			storage.SaveHeightForCommit(0, commit),
		))
		if err != nil {
			return nil, fmt.Errorf("could not bootstrap last commit & height: %w", err)
		}
	}

	var height uint64
	err = db.View(storage.RetrieveHeightByCommit(commit, &height))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last height: %w", err)
	}

	c := Core{
		db:     db,
		height: height,
		commit: commit,
	}

	return &c, nil
}

func (c *Core) Index() dps.Index {
	return &Index{core: c}
}

func (c *Core) Data() dps.Data {
	return &Data{core: c}
}

func (c *Core) Last() dps.Last {
	return &Last{core: c}
}

func (c *Core) Height() dps.Height {
	return &Height{core: c}
}

func (c *Core) Commit() dps.Commit {
	return &Commit{core: c}
}

func (c *Core) Raw() dps.Raw {
	r := Raw{
		core:   c,
		height: c.height,
	}
	return &r
}

func (c *Core) Ledger() dps.Ledger {
	l := Ledger{
		core:    c,
		version: complete.DefaultPathFinderVersion,
	}
	return &l
}

func (c *Core) payload(height uint64, path ledger.Path) (*ledger.Payload, error) {

	// Make sure that the request is for a height below the currently active
	// sentinel height; otherwise, we haven't indexed yet and we might return
	// false information because we are missing a delta.
	if height > c.height {
		return nil, fmt.Errorf("unknown height (current: %d, requested: %d)", c.height, height)
	}

	// Use seek on Ledger to seek to the next biggest key lower than the key we
	// seek for; this should represent the last update to the path before the
	// requested height and should thus be the payload we care about.
	var payload ledger.Payload
	err := c.db.View(storage.RetrievePayload(height, path, &payload))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return ledger.EmptyPayload(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve payload: %w", err)
	}

	return &payload, nil
}
