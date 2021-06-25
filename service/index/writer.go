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

package index

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

// Writer implements the `index.Writer` interface to write indexing data to
// an underlying Badger database.
type Writer struct {
	db      *badger.DB
	storage Storage
}

// NewWriter creates a new index writer that writes new indexing data to the
// given Badger database.
func NewWriter(db *badger.DB, storage Storage) *Writer {

	w := Writer{
		db:      db,
		storage: storage,
	}

	return &w
}

// First indexes the height of the first finalized block.
func (w *Writer) First(height uint64) error {
	return w.db.Update(w.storage.SaveFirst(height))
}

// Last indexes the height of the last finalized block.
func (w *Writer) Last(height uint64) error {
	return w.db.Update(w.storage.SaveLast(height))
}

// Header indexes the given header of a finalized block at the given height.
func (w *Writer) Header(height uint64, header *flow.Header) error {
	return w.db.Update(w.storage.SaveHeader(height, header))
}

// Commit indexes the given commitment of the execution state as it was after
// the execution of the finalized block at the given height.
func (w *Writer) Commit(height uint64, commit flow.StateCommitment) error {
	return w.db.Update(w.storage.SaveCommit(height, commit))
}

// Events indexes the events, which should represent all events of the finalized
// block at the given height.
func (w *Writer) Events(height uint64, events []flow.Event) error {
	buckets := make(map[flow.EventType][]flow.Event)
	for _, event := range events {
		buckets[event.Type] = append(buckets[event.Type], event)
	}
	err := w.db.Update(func(tx *badger.Txn) error {
		for typ, evts := range buckets {
			err := w.storage.SaveEvents(height, typ, evts)(tx)
			if err != nil {
				return fmt.Errorf("could not persist events: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("could not index events: %w", err)
	}
	return nil
}

// Payloads indexes the given payloads, which should represent a trie update
// of the execution state contained within the finalized block at the given
// height.
func (w *Writer) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	if len(paths) != len(payloads) {
		return fmt.Errorf("mismatch between paths and payloads counts")
	}
	return w.db.Update(func(tx *badger.Txn) error {
		for i, path := range paths {
			payload := payloads[i]
			err := w.storage.SavePayload(height, path, payload)(tx)
			if err != nil {
				return fmt.Errorf("could not save payload (path: %x): %w", path, err)
			}
		}
		return nil
	})
}

// Height indexes the height of the block.
func (w *Writer) Height(blockID flow.Identifier, height uint64) error {
	return w.db.Update(w.storage.SaveHeight(blockID, height))
}

// Transactions indexes collection IDs for the given blockID, collections themselves and the transactions that they contain.
func (w *Writer) Transactions(blockID flow.Identifier, collections []flow.LightCollection, transactions []flow.Transaction) error {
	// Index each collection by ID, and build the slice of collection IDs for the block index.
	var cIDs []flow.Identifier
	for _, collection := range collections {
		err := w.db.Update(w.storage.SaveCollection(collection))
		if err != nil {
			return fmt.Errorf("unable to save collection: %w", err)
		}

		cIDs = append(cIDs, collection.ID())
	}

	// Index each transaction by ID, and build the slice of transaction IDs for the block index.
	var tIDs []flow.Identifier
	for _, transaction := range transactions {
		err := w.db.Update(w.storage.SaveTransaction(transaction))
		if err != nil {
			return fmt.Errorf("unable to save transaction: %w", err)
		}

		tIDs = append(tIDs, transaction.ID())
	}

	// Index all collection IDs within a block.
	err := w.db.Update(w.storage.SaveCollections(blockID, cIDs))
	if err != nil {
		return fmt.Errorf("unable to save block collection list: %w", err)
	}

	// Index all transaction IDs within a block.
	err = w.db.Update(w.storage.SaveTransactions(blockID, tIDs))
	if err != nil {
		return fmt.Errorf("unable to save block transaction list: %w", err)
	}

	return nil
}
