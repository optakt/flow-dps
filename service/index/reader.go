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
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

// Reader implements the `index.Reader` interface on top of the DPS server's
// Badger database index.
type Reader struct {
	db      *badger.DB
	storage Storage
}

// NewReader creates a new index reader, using the given database as the
// underlying state repository. It is recommended to provide a read-only Badger
// database.
func NewReader(db *badger.DB, storage Storage) *Reader {

	r := Reader{
		db:      db,
		storage: storage,
	}

	return &r
}

// First returns the height of the first finalized block that was indexed.
func (r *Reader) First() (uint64, error) {
	var height uint64
	err := r.db.View(r.storage.RetrieveFirst(&height))
	return height, err
}

// Last returns the height of the last finalized block that was indexed.
func (r *Reader) Last() (uint64, error) {
	var height uint64
	err := r.db.View(r.storage.RetrieveLast(&height))
	return height, err
}

// Sealed returns the height of the last seal that was indexed.
func (r *Reader) Sealed() (uint64, error) {
	var height uint64
	err := r.db.View(r.storage.RetrieveSealed(&height))
	return height, err
}

func (r *Reader) HeightForBlock(blockID flow.Identifier) (uint64, error) {
	var height uint64
	err := r.db.View(r.storage.LookupHeightForBlock(blockID, &height))
	return height, err
}

// Commit returns the commitment of the execution state as it was after the
// execution of the finalized block at the given height.
func (r *Reader) Commit(height uint64) (flow.StateCommitment, error) {
	var commit flow.StateCommitment
	err := r.db.View(r.storage.RetrieveCommit(height, &commit))
	return commit, err
}

// Header returns the header for the finalized block at the given height.
func (r *Reader) Header(height uint64) (*flow.Header, error) {
	var header flow.Header
	err := r.db.View(r.storage.RetrieveHeader(height, &header))
	return &header, err
}

// Values returns the Ledger values of the execution state at the given paths
// as they were after the execution of the finalized block at the given height.
// For compatibility with existing Flow execution node code, a path that is not
// found within the indexed execution state returns a nil value without error.
func (r *Reader) Values(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
	first, err := r.First()
	if err != nil {
		return nil, fmt.Errorf("could not check first height: %w", err)
	}
	last, err := r.Last()
	if err != nil {
		return nil, fmt.Errorf("could not check last height: %w", err)
	}
	if height < first || height > last {
		return nil, fmt.Errorf("invalid height (given: %d, first: %d, last: %d)", height, first, last)
	}
	values := make([]ledger.Value, 0, len(paths))
	err = r.db.View(func(tx *badger.Txn) error {
		for _, path := range paths {
			var payload ledger.Payload
			err := r.storage.RetrievePayload(height, path, &payload)(tx)
			if errors.Is(err, badger.ErrKeyNotFound) {
				values = append(values, nil)
				continue
			}
			if err != nil {
				return fmt.Errorf("could not retrieve payload (path: %x): %w", path, err)
			}
			values = append(values, payload.Value)
		}
		return nil
	})
	return values, err
}

// Collection returns the collection with the given ID.
func (r *Reader) Collection(cID flow.Identifier) (*flow.LightCollection, error) {
	var collection flow.LightCollection
	err := r.db.View(r.storage.RetrieveCollection(cID, &collection))
	return &collection, err
}

// Transaction returns the transaction with the given ID.
func (r *Reader) Transaction(txID flow.Identifier) (*flow.TransactionBody, error) {
	var transaction flow.TransactionBody
	err := r.db.View(r.storage.RetrieveTransaction(txID, &transaction))
	return &transaction, err
}

func (r *Reader) HeightForTransaction(txID flow.Identifier) (uint64, error) {
	var height uint64
	err := r.db.View(r.storage.LookupHeightForTransaction(txID, &height))
	return height, err
}

// TransactionsByHeight returns the transaction IDs within the block with the given ID.
func (r *Reader) TransactionsByHeight(height uint64) ([]flow.Identifier, error) {
	var txIDs []flow.Identifier
	err := r.db.View(func(tx *badger.Txn) error {
		err := r.storage.LookupTransactionsForHeight(height, &txIDs)(tx)
		if err != nil {
			return fmt.Errorf("could not look up transactions: %w", err)
		}
		return nil
	})
	return txIDs, err
}

// Result returns the transaction result for the given transaction ID.
func (r *Reader) Result(txID flow.Identifier) (*flow.TransactionResult, error) {
	var results *flow.TransactionResult
	err := r.db.View(func(tx *badger.Txn) error {
		err := r.storage.RetrieveResult(txID, results)(tx)
		if err != nil {
			return fmt.Errorf("could not look up transaction results: %w", err)
		}
		return nil
	})
	return results, err
}

// Events returns the events of all transactions that were part of the
// finalized block at the given height. It can optionally filter them by event
// type; if no event types are given, all events are returned.
func (r *Reader) Events(height uint64, types ...flow.EventType) ([]flow.Event, error) {
	first, err := r.First()
	if err != nil {
		return nil, fmt.Errorf("could not check first height: %w", err)
	}
	last, err := r.Last()
	if err != nil {
		return nil, fmt.Errorf("could not check last height: %w", err)
	}
	if height < first || height > last {
		return nil, fmt.Errorf("invalid height (given: %d, first: %d, last: %d)", height, first, last)
	}
	var events []flow.Event
	err = r.db.View(r.storage.RetrieveEvents(height, types, &events))
	return events, err
}

// Seal returns the seal with the given ID.
func (r *Reader) Seal(sealID flow.Identifier) (*flow.Seal, error) {
	var seal flow.Seal
	err := r.db.View(r.storage.RetrieveSeal(sealID, &seal))
	return &seal, err
}

// SealsByHeight returns all of the seals that were part of the finalized block at the given height.
func (r *Reader) SealsByHeight(height uint64) ([]flow.Identifier, error) {
	var sealIDs []flow.Identifier
	err := r.db.View(func(tx *badger.Txn) error {
		err := r.storage.LookupSealsForHeight(height, &sealIDs)(tx)
		if err != nil {
			return fmt.Errorf("could not look up seals: %w", err)
		}
		return nil
	})
	return sealIDs, err
}
