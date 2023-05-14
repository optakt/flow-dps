package archive

import (
	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"
)

// Library represents something that produces operations to read/write
// from/on a DPS index database.
//
// DEPRECATED: please move methods to Library2.
type Library interface {
	ReadLibrary
	WriteLibrary
}

// ReadLibrary represents something that produces operations to read from
// a DPS index database.
//
// DEPRECATED: please move methods to ReadLibrary2.
type ReadLibrary interface {
	RetrieveFirst(height *uint64) func(*badger.Txn) error
	RetrieveLast(height *uint64) func(*badger.Txn) error

	LookupHeightForBlock(blockID flow.Identifier, height *uint64) func(*badger.Txn) error
	LookupHeightForTransaction(txID flow.Identifier, height *uint64) func(*badger.Txn) error

	RetrieveCommit(height uint64, commit *flow.StateCommitment) func(*badger.Txn) error
	RetrieveHeader(height uint64, header *flow.Header) func(*badger.Txn) error
	RetrieveEvents(height uint64, types []flow.EventType, events *[]flow.Event) func(*badger.Txn) error

	LookupTransactionsForHeight(height uint64, txIDs *[]flow.Identifier) func(*badger.Txn) error
	LookupTransactionsForCollection(collID flow.Identifier, txIDs *[]flow.Identifier) func(*badger.Txn) error
	LookupCollectionsForHeight(height uint64, collIDs *[]flow.Identifier) func(*badger.Txn) error
	LookupSealsForHeight(height uint64, sealIDs *[]flow.Identifier) func(*badger.Txn) error

	RetrieveCollection(collID flow.Identifier, collection *flow.LightCollection) func(*badger.Txn) error
	RetrieveGuarantee(collID flow.Identifier, collection *flow.CollectionGuarantee) func(*badger.Txn) error
	RetrieveTransaction(txID flow.Identifier, transaction *flow.TransactionBody) func(*badger.Txn) error
	RetrieveResult(txID flow.Identifier, result *flow.TransactionResult) func(*badger.Txn) error
	RetrieveSeal(sealID flow.Identifier, seal *flow.Seal) func(*badger.Txn) error
}

// WriteLibrary represents something that produces operations to write on
// a DPS index database.
//
// DEPRECATED: please move methods to WriteLibrary2.
type WriteLibrary interface {
	SaveFirst(height uint64) func(*badger.Txn) error
	SaveLast(height uint64) func(*badger.Txn) error

	IndexHeightForBlock(blockID flow.Identifier, height uint64) func(*badger.Txn) error
	IndexHeightForTransaction(txID flow.Identifier, height uint64) func(*badger.Txn) error

	SaveCommit(height uint64, commit flow.StateCommitment) func(*badger.Txn) error
	SaveHeader(height uint64, header *flow.Header) func(*badger.Txn) error
	SaveEvents(height uint64, typ flow.EventType, events []flow.Event) func(*badger.Txn) error

	IndexTransactionsForHeight(height uint64, txIDs []flow.Identifier) func(*badger.Txn) error
	IndexTransactionsForCollection(collID flow.Identifier, txIDs []flow.Identifier) func(*badger.Txn) error
	IndexCollectionsForHeight(height uint64, collIDs []flow.Identifier) func(*badger.Txn) error
	IndexSealsForHeight(height uint64, sealIDs []flow.Identifier) func(*badger.Txn) error

	SaveCollection(collection *flow.LightCollection) func(*badger.Txn) error
	SaveGuarantee(guarantee *flow.CollectionGuarantee) func(*badger.Txn) error
	SaveTransaction(transaction *flow.TransactionBody) func(*badger.Txn) error
	SaveResult(results *flow.TransactionResult) func(*badger.Txn) error
	SaveSeal(seal *flow.Seal) func(*badger.Txn) error
}
