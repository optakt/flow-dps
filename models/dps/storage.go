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

package dps

import (
	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Library interface {
	ReadLibrary
	WriteLibrary
}

type ReadLibrary interface {
	RetrieveFirst(height *uint64) func(*badger.Txn) error
	RetrieveLast(height *uint64) func(*badger.Txn) error

	LookupHeightForBlock(blockID flow.Identifier, height *uint64) func(*badger.Txn) error
	LookupHeightForTransaction(txID flow.Identifier, height *uint64) func(*badger.Txn) error

	RetrieveCommit(height uint64, commit *flow.StateCommitment) func(*badger.Txn) error
	RetrieveHeader(height uint64, header *flow.Header) func(*badger.Txn) error
	RetrieveEvents(height uint64, types []flow.EventType, events *[]flow.Event) func(*badger.Txn) error
	RetrievePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.Txn) error

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

type WriteLibrary interface {
	SaveFirst(height uint64) func(*badger.Txn) error
	SaveLast(height uint64) func(*badger.Txn) error

	IndexHeightForBlock(blockID flow.Identifier, height uint64) func(*badger.Txn) error
	IndexHeightForTransaction(txID flow.Identifier, height uint64) func(*badger.Txn) error

	SaveCommit(height uint64, commit flow.StateCommitment) func(*badger.Txn) error
	SaveHeader(height uint64, header *flow.Header) func(*badger.Txn) error
	SaveEvents(height uint64, typ flow.EventType, events []flow.Event) func(*badger.Txn) error
	SavePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.Txn) error

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
