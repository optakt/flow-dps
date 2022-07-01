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

//go:build integration
// +build integration

package storage_test

import (
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/service/storage"
	"github.com/onflow/flow-dps/testing/helpers"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestLibrary(t *testing.T) {
	t.Run("first", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		err := db.Update(lib.SaveFirst(mocks.GenericHeight))
		assert.NoError(t, err)

		var got uint64
		err = db.View(lib.RetrieveFirst(&got))

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("last", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		err := db.Update(lib.SaveLast(mocks.GenericHeight))
		assert.NoError(t, err)

		var got uint64
		err = db.View(lib.RetrieveLast(&got))

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("height for block", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		blockID := mocks.GenericHeader.ID()
		err := db.Update(lib.IndexHeightForBlock(blockID, mocks.GenericHeight))
		assert.NoError(t, err)

		var got uint64
		err = db.View(lib.LookupHeightForBlock(blockID, &got))

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("commit", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		err := db.Update(lib.SaveCommit(mocks.GenericHeight, mocks.GenericCommit(0)))
		assert.NoError(t, err)

		var got flow.StateCommitment
		err = db.View(lib.RetrieveCommit(mocks.GenericHeight, &got))

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericCommit(0), got)
	})

	t.Run("header", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		err := db.Update(lib.SaveHeader(mocks.GenericHeight, mocks.GenericHeader))
		assert.NoError(t, err)

		var got flow.Header
		err = db.View(lib.RetrieveHeader(mocks.GenericHeight, &got))

		require.NoError(t, err)
		assert.Equal(t, *mocks.GenericHeader, got)
	})

	t.Run("events", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		allEvents := mocks.GenericEvents(8)

		events1 := allEvents[0:4]
		events2 := allEvents[4:8]

		// First 4 events are under type 0
		err := db.Update(lib.SaveEvents(mocks.GenericHeight, mocks.GenericEventType(0), events1))
		assert.NoError(t, err)

		// Next 4 events are under type 1
		err = db.Update(lib.SaveEvents(mocks.GenericHeight, mocks.GenericEventType(1), events2))
		assert.NoError(t, err)

		t.Run("type filter matches", func(t *testing.T) {
			t.Parallel()

			var got []flow.Event
			err = db.View(lib.RetrieveEvents(mocks.GenericHeight, mocks.GenericEventTypes(1), &got))

			require.NoError(t, err)
			assert.ElementsMatch(t, events1, got)
		})

		t.Run("no type filter", func(t *testing.T) {
			t.Parallel()

			var got []flow.Event
			err = db.View(lib.RetrieveEvents(mocks.GenericHeight, []flow.EventType{}, &got))

			require.NoError(t, err)
			assert.ElementsMatch(t, allEvents, got)
		})

		t.Run("type filter matches multiple types", func(t *testing.T) {
			t.Parallel()

			var got []flow.Event
			err = db.View(lib.RetrieveEvents(mocks.GenericHeight, mocks.GenericEventTypes(4), &got))

			require.NoError(t, err)
			assert.ElementsMatch(t, allEvents, got)
		})

		t.Run("type filter does not match", func(t *testing.T) {
			t.Parallel()

			var got []flow.Event
			err = db.View(lib.RetrieveEvents(mocks.GenericHeight, []flow.EventType{mocks.GenericEventType(2)}, &got))

			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("payload", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		err := db.Update(lib.SavePayload(mocks.GenericHeight, mocks.GenericLedgerPath(0), mocks.GenericLedgerPayload(0)))
		assert.NoError(t, err)

		var got ledger.Payload
		err = db.View(lib.RetrievePayload(mocks.GenericHeight, mocks.GenericLedgerPath(0), &got))

		require.NoError(t, err)
		assert.Equal(t, *mocks.GenericLedgerPayload(0), got)
	})

	t.Run("transaction", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		tx := mocks.GenericTransaction(0)

		err := db.Update(lib.SaveTransaction(tx))
		assert.NoError(t, err)

		var got flow.TransactionBody
		err = db.View(lib.RetrieveTransaction(tx.ID(), &got))

		require.NoError(t, err)
		assert.Equal(t, *tx, got)
	})

	t.Run("collection", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		collection := mocks.GenericCollection(0)

		err := db.Update(lib.SaveCollection(collection))
		assert.NoError(t, err)

		var got flow.LightCollection
		err = db.View(lib.RetrieveCollection(collection.ID(), &got))

		require.NoError(t, err)
		assert.Equal(t, *collection, got)
	})

	t.Run("transactions for height", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		txIDs := mocks.GenericTransactionIDs(4)

		err := db.Update(lib.IndexTransactionsForHeight(mocks.GenericHeight, txIDs))
		assert.NoError(t, err)

		var got []flow.Identifier
		err = db.View(lib.LookupTransactionsForHeight(mocks.GenericHeight, &got))

		require.NoError(t, err)
		assert.Equal(t, txIDs, got)
	})

	t.Run("transactions for collection", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		txIDs := mocks.GenericTransactionIDs(4)
		collID := mocks.GenericCollection(0).ID()

		err := db.Update(lib.IndexTransactionsForCollection(collID, txIDs))
		assert.NoError(t, err)

		var got []flow.Identifier
		err = db.View(lib.LookupTransactionsForCollection(collID, &got))

		require.NoError(t, err)
		assert.ElementsMatch(t, txIDs, got)
	})

	t.Run("collections for height", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		collIDs := mocks.GenericCollectionIDs(4)

		err := db.Update(lib.IndexCollectionsForHeight(mocks.GenericHeight, collIDs))
		assert.NoError(t, err)

		var got []flow.Identifier
		err = db.View(lib.LookupCollectionsForHeight(mocks.GenericHeight, &got))

		require.NoError(t, err)
		assert.ElementsMatch(t, collIDs, got)
	})

	t.Run("result", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		result := mocks.GenericResult(0)

		err := db.Update(lib.SaveResult(result))
		assert.NoError(t, err)

		var got flow.TransactionResult
		err = db.View(lib.RetrieveResult(result.TransactionID, &got))

		assert.NoError(t, err)
		assert.Equal(t, *result, got)
	})

	t.Run("guarantee", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		guarantee := mocks.GenericGuarantee(0)

		err := db.Update(lib.SaveGuarantee(guarantee))
		assert.NoError(t, err)

		var got flow.CollectionGuarantee
		err = db.View(lib.RetrieveGuarantee(guarantee.CollectionID, &got))

		assert.NoError(t, err)
		assert.Equal(t, *guarantee, got)
	})

	t.Run("seal", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		seal := mocks.GenericSeal(0)

		err := db.Update(lib.SaveSeal(seal))
		assert.NoError(t, err)

		var got flow.Seal
		err = db.View(lib.RetrieveSeal(seal.ID(), &got))

		assert.NoError(t, err)
		assert.Equal(t, *seal, got)
	})

	t.Run("seals by height", func(t *testing.T) {
		t.Parallel()

		db, lib := setupLibrary(t)

		sealIDs := mocks.GenericSealIDs(4)

		err := db.Update(lib.IndexSealsForHeight(mocks.GenericHeight, sealIDs))
		assert.NoError(t, err)

		var got []flow.Identifier
		err = db.View(lib.LookupSealsForHeight(mocks.GenericHeight, &got))

		assert.NoError(t, err)
		assert.ElementsMatch(t, sealIDs, got)
	})
}

func setupLibrary(t *testing.T) (*badger.DB, *storage.Library) {
	t.Helper()

	codec := zbor.NewCodec()

	return helpers.InMemoryDB(t), storage.New(codec)
}
