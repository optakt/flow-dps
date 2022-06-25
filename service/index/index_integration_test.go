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

package index_test

import (
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/service/index"
	"github.com/onflow/flow-dps/service/storage"
	"github.com/onflow/flow-dps/testing/helpers"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestIndex(t *testing.T) {
	t.Run("first", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		assert.NoError(t, writer.First(mocks.GenericHeight))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		got, err := reader.First()

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("last", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		assert.NoError(t, writer.Last(mocks.GenericHeight))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		got, err := reader.Last()

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("height", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		blockID := mocks.GenericHeader.ID()
		assert.NoError(t, writer.Height(blockID, mocks.GenericHeight))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		got, err := reader.HeightForBlock(blockID)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("commit", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		assert.NoError(t, writer.Commit(mocks.GenericHeight, mocks.GenericCommit(0)))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		got, err := reader.Commit(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericCommit(0), got)
	})

	t.Run("header", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		assert.NoError(t, writer.Header(mocks.GenericHeight, mocks.GenericHeader))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		got, err := reader.Header(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeader, got)
	})

	t.Run("payloads", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		paths := mocks.GenericLedgerPaths(4)
		payloads := mocks.GenericLedgerPayloads(4)
		values := mocks.GenericLedgerValues(4)

		assert.NoError(t, writer.First(mocks.GenericHeight))
		assert.NoError(t, writer.Last(mocks.GenericHeight))
		assert.NoError(t, writer.Payloads(mocks.GenericHeight, paths, payloads))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		got, err := reader.Values(mocks.GenericHeight, paths)

		require.NoError(t, err)
		assert.ElementsMatch(t, values, got)
	})

	t.Run("collections", func(t *testing.T) {
		t.Parallel()

		collections := mocks.GenericCollections(4)

		reader, writer, db := setupIndex(t)
		defer db.Close()

		assert.NoError(t, writer.Collections(mocks.GenericHeight, collections))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		// NOTE: The following subtests should NOT be run in parallel, because of the deferral
		// to close the database above.
		t.Run("retrieve collection by ID", func(t *testing.T) {
			got, err := reader.Collection(collections[0].ID())

			require.NoError(t, err)
			assert.Equal(t, collections[0], got)
		})

		t.Run("retrieve collections by height", func(t *testing.T) {
			got, err := reader.CollectionsByHeight(mocks.GenericHeight)

			require.NoError(t, err)
			assert.ElementsMatch(t, mocks.GenericCollectionIDs(4), got)
		})

		t.Run("retrieve transactions from collection", func(t *testing.T) {
			// For now this index is not used.
		})
	})

	t.Run("guarantees", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		assert.NoError(t, writer.Guarantees(mocks.GenericHeight, mocks.GenericGuarantees(4)))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		guarantee := mocks.GenericGuarantee(0)
		got, err := reader.Guarantee(guarantee.ID())

		require.NoError(t, err)
		assert.Equal(t, guarantee, got)
	})

	t.Run("transactions", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		transactions := mocks.GenericTransactions(4)
		txIDs := []flow.Identifier{
			transactions[0].ID(),
			transactions[1].ID(),
			transactions[2].ID(),
			transactions[3].ID(),
		}

		assert.NoError(t, writer.Transactions(mocks.GenericHeight, transactions))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		// NOTE: The following subtests should NOT be run in parallel, because of the deferral
		// to close the database above.
		t.Run("retrieve transactions by height", func(t *testing.T) {
			gotTxIDs, err := reader.TransactionsByHeight(mocks.GenericHeight)

			require.NoError(t, err)
			assert.ElementsMatch(t, txIDs, gotTxIDs)
		})

		t.Run("retrieve transaction by ID", func(t *testing.T) {
			gotTx, err := reader.Transaction(transactions[0].ID())

			require.NoError(t, err)
			assert.Equal(t, transactions[0], gotTx)
		})

		t.Run("retrieve height for transaction", func(t *testing.T) {
			gotTx, err := reader.HeightForTransaction(transactions[0].ID())

			require.NoError(t, err)
			assert.Equal(t, mocks.GenericHeight, gotTx)
		})
	})

	t.Run("results", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		results := mocks.GenericResults(4)

		assert.NoError(t, writer.Results(results))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		got, err := reader.Result(results[0].TransactionID)

		require.NoError(t, err)
		assert.Equal(t, results[0], got)
	})

	t.Run("events", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		withdrawalType := mocks.GenericEventType(0)
		depositType := mocks.GenericEventType(1)
		withdrawals := mocks.GenericEvents(2, withdrawalType)
		deposits := mocks.GenericEvents(2, depositType)
		events := append(withdrawals, deposits...)

		assert.NoError(t, writer.First(mocks.GenericHeight))
		assert.NoError(t, writer.Last(mocks.GenericHeight))
		assert.NoError(t, writer.Events(mocks.GenericHeight, events))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		// NOTE: The following subtests should NOT be run in parallel, because of the deferral
		// to close the database above.
		t.Run("no types specified", func(t *testing.T) {
			got, err := reader.Events(mocks.GenericHeight)

			require.NoError(t, err)
			assert.ElementsMatch(t, events, got)
		})

		t.Run("type specified", func(t *testing.T) {
			got1, err := reader.Events(mocks.GenericHeight, withdrawalType)

			require.NoError(t, err)
			assert.Len(t, got1, 2)

			got2, err := reader.Events(mocks.GenericHeight, depositType)

			require.NoError(t, err)
			assert.Len(t, got1, 2)

			assert.NotEqual(t, got1, got2)
		})
	})

	t.Run("seals", func(t *testing.T) {
		t.Parallel()

		reader, writer, db := setupIndex(t)
		defer db.Close()

		seals := mocks.GenericSeals(4)

		assert.NoError(t, writer.Seals(mocks.GenericHeight, seals))
		// Close the writer to make it commit its transactions.
		require.NoError(t, writer.Close())

		// NOTE: The following subtests should NOT be run in parallel, because of the deferral
		// to close the database above.
		t.Run("retrieve seal by ID", func(t *testing.T) {
			got, err := reader.Seal(seals[0].ID())

			require.NoError(t, err)
			assert.Equal(t, seals[0], got)
		})

		t.Run("retrieve seals by height", func(t *testing.T) {
			got, err := reader.SealsByHeight(mocks.GenericHeight)

			require.NoError(t, err)
			assert.ElementsMatch(t, got, mocks.GenericSealIDs(4))
		})
	})
}

func setupIndex(t *testing.T) (*index.Reader, *index.Writer, *badger.DB) {
	t.Helper()

	codec := zbor.NewCodec()

	db := helpers.InMemoryDB(t)

	lib := storage.New(codec)

	reader := index.NewReader(db, lib)
	writer := index.NewWriter(db, lib, index.WithConcurrentTransactions(4))

	return reader, writer, db
}
