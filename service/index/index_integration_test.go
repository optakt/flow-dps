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

// +build integration

package index_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/service/storage"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestIndex(t *testing.T) {

	t.Run("first", func(t *testing.T) {
		t.Parallel()

		reader, writer := setupIndex(t)

		assert.NoError(t, writer.First(mocks.GenericHeight))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		got, err := reader.First()

		assert.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("last", func(t *testing.T) {
		t.Parallel()

		reader, writer := setupIndex(t)

		assert.NoError(t, writer.Last(mocks.GenericHeight))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		got, err := reader.Last()

		assert.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("height", func(t *testing.T) {
		t.Parallel()

		reader, writer := setupIndex(t)

		assert.NoError(t, writer.Height(mocks.GenericIdentifier(0), mocks.GenericHeight))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		got, err := reader.HeightForBlock(mocks.GenericIdentifier(0))

		assert.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("commit", func(t *testing.T) {
		t.Parallel()

		reader, writer := setupIndex(t)

		assert.NoError(t, writer.Commit(mocks.GenericHeight, mocks.GenericCommit(0)))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		got, err := reader.Commit(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, mocks.GenericCommit(0), got)
	})

	t.Run("header", func(t *testing.T) {
		t.Parallel()

		reader, writer := setupIndex(t)

		assert.NoError(t, writer.Header(mocks.GenericHeight, mocks.GenericHeader))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		got, err := reader.Header(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, mocks.GenericHeader, got)
	})

	t.Run("payloads", func(t *testing.T) {
		t.Parallel()

		reader, writer := setupIndex(t)

		paths := mocks.GenericLedgerPaths(4)
		payloads := mocks.GenericLedgerPayloads(4)

		assert.NoError(t, writer.First(mocks.GenericHeight))
		assert.NoError(t, writer.Last(mocks.GenericHeight))
		assert.NoError(t, writer.Payloads(mocks.GenericHeight, paths, payloads))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		got, err := reader.Values(mocks.GenericHeight, paths)

		assert.NoError(t, err)
		assert.ElementsMatch(t, mocks.GenericLedgerValues(4), got)
	})

	t.Run("collections", func(t *testing.T) {
		t.Parallel()

		_, writer := setupIndex(t)

		assert.NoError(t, writer.Collections(mocks.GenericHeight, mocks.GenericCollections(4)))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		// TODO: Once https://github.com/optakt/flow-dps/pull/301 is merged
	})

	t.Run("transactions", func(t *testing.T) {
		t.Parallel()

		reader, writer := setupIndex(t)

		transactions := mocks.GenericTransactions(4)
		txIDs := []flow.Identifier{
			transactions[0].ID(),
			transactions[1].ID(),
			transactions[2].ID(),
			transactions[3].ID(),
		}

		assert.NoError(t, writer.Transactions(mocks.GenericHeight, transactions))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		gotTxIDs, err := reader.TransactionsByHeight(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.ElementsMatch(t, txIDs, gotTxIDs)

		gotTx, err := reader.Transaction(transactions[0].ID())

		assert.NoError(t, err)
		assert.Equal(t, transactions[0], gotTx)
	})

	t.Run("events", func(t *testing.T) {
		t.Parallel()

		reader, writer := setupIndex(t)

		events := mocks.GenericEvents(4)

		assert.NoError(t, writer.First(mocks.GenericHeight))
		assert.NoError(t, writer.Last(mocks.GenericHeight))
		assert.NoError(t, writer.Events(mocks.GenericHeight, events))
		// Close the writer to make it commit its transactions.
		assert.NoError(t, writer.Close())

		t.Run("no types specified", func(t *testing.T) {
			got, err := reader.Events(mocks.GenericHeight)

			assert.NoError(t, err)
			assert.ElementsMatch(t, events, got)
		})

		t.Run("type specified", func(t *testing.T) {
			got1, err := reader.Events(mocks.GenericHeight, mocks.GenericEventType(0))

			assert.NoError(t, err)
			assert.Len(t, got1, 2)

			got2, err := reader.Events(mocks.GenericHeight, mocks.GenericEventType(1))

			assert.NoError(t, err)
			assert.Len(t, got1, 2)

			assert.NotEqual(t, got1, got2)
		})
	})
}

func setupIndex(t *testing.T) (*index.Reader, *index.Writer) {
	t.Helper()

	codec, err := zbor.NewCodec()
	require.NoError(t, err)

	db := helpers.InMemoryDB(t)
	lib := storage.New(codec)

	reader := index.NewReader(db, lib)
	writer := index.NewWriter(db, lib, index.WithConcurrentTransactions(4))

	return reader, writer
}
