package storage

import (
	"encoding/binary"
	"fmt"

	"github.com/OneOfOne/xxhash"
	"github.com/dgraph-io/badger/v2"

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

// BatchSavePayload is an operation that writes the height of a slice of paths and a slice of payloads.
func (l *Library) BatchSavePayload(height uint64, payload flow.RegisterEntry) func(*badger.WriteBatch) error {
	return l.batchWrite(EncodeKey(PrefixPayload, payload.Key, height), payload.Value)
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

// RetrievePayload retrieves the ledger payloads at the given height that match the given registerID.
func (l *Library) RetrievePayload(height uint64, register flow.RegisterID, payload *flow.RegisterValue) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		key := EncodeKey(PrefixPayload, register, height)
		it := tx.NewIterator(badger.IteratorOptions{
			PrefetchSize:   0,
			PrefetchValues: false,
			Reverse:        true,
			AllVersions:    false,
			InternalAccess: false,
			Prefix:         key[:len(key)-8],
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
