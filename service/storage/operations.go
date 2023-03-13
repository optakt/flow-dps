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

package storage

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/OneOfOne/xxhash"
	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/model/flow"
)

// SaveFirst is an operation that writes the height of the first indexed block.
func (l *Library) SaveFirst(height uint64) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixFirst), height)
}

// SaveLast is an operation that writes the height of the last indexed block.
func (l *Library) SaveLast(height uint64) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixLast), height)
}

// IndexHeightForBlock is an operation that indexes the given height for its block identifier.
func (l *Library) IndexHeightForBlock(blockID flow.Identifier, height uint64) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixHeightForBlock, blockID), height)
}

// SaveCommit is an operation that writes the height of a state commitment.
func (l *Library) SaveCommit(height uint64, commit flow.StateCommitment) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixCommit, height), commit)
}

// SaveHeader is an operation that writes the height of a header.
func (l *Library) SaveHeader(height uint64, header *flow.Header) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixHeader, height), header)
}

// SaveEvents is an operation that writes the height and type of a slice of events.
func (l *Library) SaveEvents(height uint64, typ flow.EventType, events []flow.Event) func(*badger.Txn) error {
	hash := xxhash.ChecksumString64(string(typ))
	return l.save(EncodeKey(PrefixEvents, height, hash), events)
}

// SavePayload is an operation that writes the height of a slice of paths and a slice of payloads.
func (l *Library) SavePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixPayload, path, height), payload)
}

// BatchSavePayload is an operation that writes the height of a slice of paths and a slice of payloads.
func (l *Library) BatchSavePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.WriteBatch) error {
	return l.batchWrite(EncodeKey(PrefixPayload, path, height), payload)
}

// SaveTransaction is an operation that writes the given transaction.
func (l *Library) SaveTransaction(transaction *flow.TransactionBody) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixTransaction, transaction.ID()), transaction)
}

// IndexHeightForTransaction is an operation that writes the height a transaction identifier.
func (l *Library) IndexHeightForTransaction(txID flow.Identifier, height uint64) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixHeightForTransaction, txID), height)
}

// SaveCollection is an operation that writes the given collection.
func (l *Library) SaveCollection(collection *flow.LightCollection) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixCollection, collection.ID()), collection)
}

// SaveGuarantee is an operation that writes the given guarantee.
func (l *Library) SaveGuarantee(guarantee *flow.CollectionGuarantee) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixGuarantee, guarantee.CollectionID), guarantee)
}

// SaveSeal is an operation that writes the given seal.
func (l *Library) SaveSeal(seal *flow.Seal) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixSeal, seal.ID()), seal)
}

// IndexTransactionsForHeight is an operation that indexes the height of a slice of transaction identifiers.
func (l *Library) IndexTransactionsForHeight(height uint64, txIDs []flow.Identifier) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixTransactionsForHeight, height), txIDs)
}

// IndexTransactionsForCollection is an operation that indexes the collection identifier to which a slice
// of transactions belongs.
func (l *Library) IndexTransactionsForCollection(collID flow.Identifier, txIDs []flow.Identifier) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixTransactionsForCollection, collID), txIDs)
}

// IndexCollectionsForHeight is an operation that indexes the height of a slice of collection identifiers.
func (l *Library) IndexCollectionsForHeight(height uint64, collIDs []flow.Identifier) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixCollectionsForHeight, height), collIDs)
}

// IndexSealsForHeight is an operation that indexes the height of a slice of seal identifiers.
func (l *Library) IndexSealsForHeight(height uint64, sealIDs []flow.Identifier) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixSealsForHeight, height), sealIDs)
}

// SaveResult is an operation that writes the given transaction result.
func (l *Library) SaveResult(result *flow.TransactionResult) func(*badger.Txn) error {
	return l.save(EncodeKey(PrefixResults, result.TransactionID), result)
}

// RetrieveFirst retrieves the first indexed height.
func (l *Library) RetrieveFirst(height *uint64) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixFirst), height)
}

// RetrieveLast retrieves the last indexed height.
func (l *Library) RetrieveLast(height *uint64) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixLast), height)
}

// LookupHeightForBlock retrieves the height of the given block identifier.
func (l *Library) LookupHeightForBlock(blockID flow.Identifier, height *uint64) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixHeightForBlock, blockID), height)
}

// RetrieveHeader retrieves the header at the given height.
func (l *Library) RetrieveHeader(height uint64, header *flow.Header) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixHeader, height), header)
}

// RetrieveCommit retrieves the commit at the given height.
func (l *Library) RetrieveCommit(height uint64, commit *flow.StateCommitment) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixCommit, height), commit)
}

// RetrieveEvents retrieves the events at the given height that match with the specified types.
// If no types were provided, all events are retrieved.
func (l *Library) RetrieveEvents(height uint64, types []flow.EventType, events *[]flow.Event) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		lookup := make(map[uint64]struct{})
		for _, typ := range types {
			hash := xxhash.ChecksumString64(string(typ))
			lookup[hash] = struct{}{}
		}

		prefix := EncodeKey(PrefixEvents, height)
		opts := badger.DefaultIteratorOptions
		// NOTE: this is an optimization only, it does not enforce that all
		// results in the iteration have this prefix.
		opts.Prefix = prefix

		it := tx.NewIterator(opts)
		defer it.Close()

		// Iterate on all keys with the right prefix.
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			// If types were given for filtering, discard events which should not be included.
			hash := binary.BigEndian.Uint64(it.Item().Key()[1+8:])
			_, ok := lookup[hash]
			if len(lookup) != 0 && !ok {
				continue
			}

			// Unmarshal event batch and append them to result slice.
			var evts []flow.Event
			err := it.Item().Value(func(val []byte) error {
				return l.codec.Unmarshal(val, &evts)
			})
			if err != nil {
				return fmt.Errorf("could not unmarshal events: %w", err)
			}

			*events = append(*events, evts...)
		}

		return nil
	}
}

// RetrievePayload retrieves the ledger payloads at the given height that match the given path.
func (l *Library) RetrievePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {

		key := EncodeKey(PrefixPayload, path, height)
		it := tx.NewIterator(badger.IteratorOptions{
			PrefetchSize:   0,
			PrefetchValues: false,
			Reverse:        true,
			AllVersions:    false,
			InternalAccess: false,
			Prefix:         key[:1+pathfinder.PathByteSize],
		})
		defer it.Close()

		it.Seek(key)
		if !it.Valid() {
			return badger.ErrKeyNotFound
		}

		err := it.Item().Value(func(val []byte) error {
			return l.codec.Unmarshal(val, payload)
		})

		return err
	}
}

// RetrieveCollection retrieves the collection with the given identifier.
func (l *Library) RetrieveCollection(collectionID flow.Identifier, collection *flow.LightCollection) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixCollection, collectionID), collection)
}

// RetrieveGuarantee retrieves the guarantee with the given collection identifier.
func (l *Library) RetrieveGuarantee(collectionID flow.Identifier, guarantee *flow.CollectionGuarantee) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixGuarantee, collectionID), guarantee)
}

// RetrieveTransaction retrieves the transaction with the given identifier.
func (l *Library) RetrieveTransaction(transactionID flow.Identifier, transaction *flow.TransactionBody) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixTransaction, transactionID), transaction)
}

// LookupHeightForTransaction retrieves the height of the transaction with the given identifier.
func (l *Library) LookupHeightForTransaction(txID flow.Identifier, height *uint64) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixHeightForTransaction, txID), height)
}

// RetrieveSeal retrieves the seal with the given identifier.
func (l *Library) RetrieveSeal(sealID flow.Identifier, seal *flow.Seal) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixSeal, sealID), seal)
}

// LookupCollectionsForHeight retrieves the identifiers of collections at the given height.
func (l *Library) LookupCollectionsForHeight(height uint64, collIDs *[]flow.Identifier) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixCollectionsForHeight, height), collIDs)
}

// LookupTransactionsForHeight retrieves the identifiers of transactions at the given height.
func (l *Library) LookupTransactionsForHeight(height uint64, txIDs *[]flow.Identifier) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixTransactionsForHeight, height), txIDs)
}

// LookupTransactionsForCollection retrieves the identifiers of transactions within the collection
// with the given identifier.
func (l *Library) LookupTransactionsForCollection(collID flow.Identifier, txIDs *[]flow.Identifier) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixTransactionsForCollection, collID), txIDs)
}

// LookupSealsForHeight retrieves the identifiers of seals at the given height.
func (l *Library) LookupSealsForHeight(height uint64, sealIDs *[]flow.Identifier) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixSealsForHeight, height), sealIDs)
}

// RetrieveResult retrieves the result with the given transaction identifier.
func (l *Library) RetrieveResult(txID flow.Identifier, result *flow.TransactionResult) func(*badger.Txn) error {
	return l.retrieve(EncodeKey(PrefixResults, txID), result)
}

// IterateLedger steps through the entire ledger for ledger keys and payloads
// and call the given callback for each of them.
func (l *Library) IterateLedger(exclude func(height uint64) bool, process func(path ledger.Path, payload *ledger.Payload) error) func(*badger.Txn) error {

	prefix := EncodeKey(PrefixPayload)
	opts := badger.IteratorOptions{
		PrefetchSize:   100,
		PrefetchValues: false,
		Reverse:        true,
		AllVersions:    false,
		InternalAccess: false,
		Prefix:         prefix,
	}
	highest := ledger.Path{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}

	return func(tx *badger.Txn) error {

		it := tx.NewIterator(opts)
		defer it.Close()

		sentinel := EncodeKey(PrefixPayload, highest, uint64(math.MaxUint64))
		for it.Seek(sentinel); it.ValidForPrefix(prefix); {

			// First, we extract the height from the item's key, and check if
			// we should just skip past this entry.
			item := it.Item()
			key := item.Key()
			height := binary.BigEndian.Uint64(key[33:41])
			if exclude(height) {
				it.Next()
				continue
			}

			// Next, we can get the path from the key and the payload from the
			// value.
			var path ledger.Path
			var payload ledger.Payload
			copy(path[:], key[1:33])
			err := item.Value(func(val []byte) error {
				err := l.codec.Unmarshal(val, &payload)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("could not decode value (path: %x): %w", path, err)
			}

			// Then, we process the ledger path and payload with the callback.
			err = process(path, &payload)
			if err != nil {
				return fmt.Errorf("could not process register (path: %x): %w", path, err)
			}

			// We need want to go to the first value that is below the current
			// path. In order to cover all potential cases, including payloads
			// at height zero, we need to decrement the current path by one and
			// use the maximum possible height. If the decrement doesn't work,
			// we have reached the zero path and we can break; otherwise, we
			// would just wrap around to the maximum key again.
			var zero ledger.Path
			if path == zero {
				break
			}
			for i := len(path) - 1; i >= 0; i-- {
				path[i] = path[i] - 1
				if path[i] != 0xff {
					break
				}
			}
			sentinel = EncodeKey(PrefixPayload, path, uint64(math.MaxUint64))
			it.Seek(sentinel)
		}

		return nil
	}
}
