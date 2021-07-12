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

	"github.com/OneOfOne/xxhash"
	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/model/flow"
)

func (l *Library) SaveFirst(height uint64) func(*badger.Txn) error {
	return l.save(encodeKey(prefixFirst), height)
}

func (l *Library) SaveLast(height uint64) func(*badger.Txn) error {
	return l.save(encodeKey(prefixLast), height)
}

func (l *Library) IndexHeightForBlock(blockID flow.Identifier, height uint64) func(*badger.Txn) error {
	return l.save(encodeKey(prefixHeightForBlock, blockID), height)
}

func (l *Library) SaveCommit(height uint64, commit flow.StateCommitment) func(*badger.Txn) error {
	return l.save(encodeKey(prefixCommit, height), commit)
}

func (l *Library) SaveHeader(height uint64, header *flow.Header) func(*badger.Txn) error {
	return l.save(encodeKey(prefixHeader, height), header)
}

func (l *Library) SaveEvents(height uint64, typ flow.EventType, events []flow.Event) func(*badger.Txn) error {
	hash := xxhash.ChecksumString64(string(typ))
	return l.save(encodeKey(prefixEvents, height, hash), events)
}

func (l *Library) SavePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.Txn) error {
	return l.save(encodeKey(prefixPayload, path, height), payload)
}

func (l *Library) SaveTransaction(transaction *flow.TransactionBody) func(*badger.Txn) error {
	return l.save(encodeKey(prefixTransaction, transaction.ID()), transaction)
}

func (l *Library) SaveCollection(collection *flow.LightCollection) func(*badger.Txn) error {
	return l.save(encodeKey(prefixCollection, collection.ID()), collection)
}

func (l *Library) IndexTransactionsForHeight(height uint64, txIDs []flow.Identifier) func(*badger.Txn) error {
	return l.save(encodeKey(prefixTransactionsForHeight, height), txIDs)
}

func (l *Library) IndexTransactionsForCollection(collID flow.Identifier, txIDs []flow.Identifier) func(*badger.Txn) error {
	return l.save(encodeKey(prefixTransactionsForCollection, collID), txIDs)
}

func (l *Library) IndexCollectionsForHeight(height uint64, collIDs []flow.Identifier) func(*badger.Txn) error {
	return l.save(encodeKey(prefixCollectionsForHeight, height), collIDs)
}

func (l *Library) RetrieveFirst(height *uint64) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixFirst), height)
}

func (l *Library) RetrieveLast(height *uint64) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixLast), height)
}

func (l *Library) LookupHeightForBlock(blockID flow.Identifier, height *uint64) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixHeightForBlock, blockID), height)
}

func (l *Library) RetrieveHeader(height uint64, header *flow.Header) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixHeader, height), header)
}

func (l *Library) RetrieveCommit(height uint64, commit *flow.StateCommitment) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixCommit, height), commit)
}

func (l *Library) RetrieveEvents(height uint64, types []flow.EventType, events *[]flow.Event) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		lookup := make(map[uint64]struct{})
		for _, typ := range types {
			hash := xxhash.ChecksumString64(string(typ))
			lookup[hash] = struct{}{}
		}

		prefix := encodeKey(prefixEvents, height)
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

func (l *Library) RetrievePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		key := encodeKey(prefixPayload, path, height)
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

func (l *Library) RetrieveCollection(collectionID flow.Identifier, collection *flow.LightCollection) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixCollection, collectionID), collection)
}

func (l *Library) RetrieveTransaction(transactionID flow.Identifier, transaction *flow.TransactionBody) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixTransaction, transactionID), transaction)
}

func (l *Library) LookupCollectionsForHeight(height uint64, collIDs *[]flow.Identifier) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixCollectionsForHeight, height), collIDs)
}

func (l *Library) LookupTransactionsForHeight(height uint64, txIDs *[]flow.Identifier) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixTransactionsForHeight, height), txIDs)
}

func (l *Library) LookupTransactionsForCollection(collID flow.Identifier, txIDs *[]flow.Identifier) func(*badger.Txn) error {
	return l.retrieve(encodeKey(prefixTransactionsForCollection, collID), txIDs)
}
