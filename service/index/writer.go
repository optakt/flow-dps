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
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v2"
	"golang.org/x/sync/semaphore"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

// Writer implements the `index.Writer` interface to write indexing data to
// an underlying Badger database.
type Writer struct {
	sync.RWMutex
	db   *badger.DB
	lib  dps.WriteLibrary
	cfg  Config
	tx   *badger.Txn
	sema *semaphore.Weighted
	err  error
}

// NewWriter creates a new index writer that writes new indexing data to the
// given Badger database.
func NewWriter(db *badger.DB, lib dps.WriteLibrary, options ...func(*Config)) *Writer {

	cfg := DefaultConfig
	for _, option := range options {
		option(&cfg)
	}

	w := Writer{
		db:   db,
		lib:  lib,
		cfg:  cfg,
		tx:   db.NewTransaction(true),
		sema: semaphore.NewWeighted(int64(cfg.ConcurrentTransactions)),
		err:  nil,
	}

	return &w
}

// First indexes the height of the first finalized block.
func (w *Writer) First(height uint64) error {
	return w.apply(w.lib.SaveFirst(height))
}

// Last indexes the height of the last finalized block.
func (w *Writer) Last(height uint64) error {
	return w.apply(w.lib.SaveLast(height))
}

// Height indexes the height for the given block ID.
func (w *Writer) Height(blockID flow.Identifier, height uint64) error {
	return w.apply(w.lib.IndexHeightForBlock(blockID, height))
}

// Commit indexes the given commitment of the execution state as it was after
// the execution of the finalized block at the given height.
func (w *Writer) Commit(height uint64, commit flow.StateCommitment) error {
	return w.apply(w.lib.SaveCommit(height, commit))
}

// Header indexes the given header of a finalized block at the given height.
func (w *Writer) Header(height uint64, header *flow.Header) error {
	return w.apply(w.lib.SaveHeader(height, header))
}

// Payloads indexes the given payloads, which should represent a trie update
// of the execution state contained within the finalized block at the given
// height.
func (w *Writer) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	if len(paths) != len(payloads) {
		return fmt.Errorf("mismatch between paths and payloads counts")
	}
	return w.apply(func(tx *badger.Txn) error {
		for i, path := range paths {
			payload := payloads[i]
			err := w.lib.SavePayload(height, path, payload)(tx)
			if err != nil {
				return fmt.Errorf("could not save payload (path: %x): %w", path, err)
			}
		}
		return nil
	})
}

func (w *Writer) Collections(height uint64, collections []*flow.LightCollection) error {
	var collIDs []flow.Identifier
	return w.apply(func(tx *badger.Txn) error {
		for _, collection := range collections {
			err := w.lib.SaveCollection(collection)(tx)
			if err != nil {
				return fmt.Errorf("could not store collection (id: %x): %w", collection.ID(), err)
			}
			collID := collection.ID()
			err = w.lib.IndexTransactionsForCollection(collID, collection.Transactions)(tx)
			if err != nil {
				return fmt.Errorf("could not index transactions for collection (id: %x): %w", collID, err)
			}
			collIDs = append(collIDs, collID)
		}
		err := w.lib.IndexCollectionsForHeight(height, collIDs)(tx)
		if err != nil {
			return fmt.Errorf("could not index collections for height: %w", err)
		}
		return nil
	})
}

func (w *Writer) Transactions(height uint64, transactions []*flow.TransactionBody) error {
	var txIDs []flow.Identifier
	return w.apply(func(tx *badger.Txn) error {
		for _, transaction := range transactions {
			err := w.lib.SaveTransaction(transaction)(tx)
			if err != nil {
				return fmt.Errorf("could not save transaction (id: %x): %w", transaction.ID(), err)
			}
			txIDs = append(txIDs, transaction.ID())
		}
		err := w.lib.IndexTransactionsForHeight(height, txIDs)(tx)
		if err != nil {
			return fmt.Errorf("could not index transactions for height: %w", err)
		}
		return nil
	})
}

// Events indexes the events, which should represent all events of the finalized
// block at the given height.
func (w *Writer) Events(height uint64, events []flow.Event) error {
	buckets := make(map[flow.EventType][]flow.Event)
	for _, event := range events {
		buckets[event.Type] = append(buckets[event.Type], event)
	}
	err := w.apply(func(tx *badger.Txn) error {
		for typ, evts := range buckets {
			err := w.lib.SaveEvents(height, typ, evts)(tx)
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

func (w *Writer) apply(op func(*badger.Txn) error) error {

	// Before applying an additional operation to the transaction we are
	// currently building, we want to see if there was an error committing the
	// previous transaction. As it is set in a callback, we copy the error while
	// guarded with a read lock.
	w.RLock()
	err := w.err
	w.RUnlock()
	if err != nil {
		return fmt.Errorf("could not commit transaction: %w", w.err)
	}

	// If we had no error in a previous transaction, we try applying the
	// operation to the current transaction. If the transaction is already too
	// big, we simply commit it with our callback and start a new transaction.
	// Transaction creation is guarded by a semaphore that limits it to the
	// configured number of inflight transactions.
	err = op(w.tx)
	if errors.Is(err, badger.ErrTxnTooBig) {
		w.tx.CommitWith(w.done)
		_ = w.sema.Acquire(context.Background(), 1)
		w.tx = w.db.NewTransaction(true)
		err = op(w.tx)
	}
	if err != nil {
		return fmt.Errorf("could not apply operation: %w", err)
	}

	return nil
}

func (w *Writer) done(err error) {

	// When a transaction is fully committed, we get the result in this
	// callback. If we have an error, we acquire the write lock for the error
	// and store it.
	if err != nil {
		w.Lock()
		w.err = err
		w.Unlock()
	}

	// Releasing one resource on the semaphore will free up one slot for
	// inflight transactions.
	w.sema.Release(1)
}

func (w *Writer) Close() error {

	// When closing the writer, we should no longer be applying operations. This
	// means we only have to wait for all inflight transactions to commit. This
	// is guaranteed if we are able to acquire all of the resources on the
	// semaphore, which we do here.
	_ = w.sema.Acquire(context.Background(), int64(w.cfg.ConcurrentTransactions))
	if w.err != nil {
		return fmt.Errorf("could not flush transactions: %w", w.err)
	}

	return nil
}
