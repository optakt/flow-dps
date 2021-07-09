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
	"testing"

	"github.com/OneOfOne/xxhash"
	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestLibrary_SaveAndRetrieveFirst(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testKey := encodeKey(prefixFirst)

	t.Run("save first height", func(t *testing.T) {

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, uint64(0), v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveFirst(mocks.GenericHeight))
		assert.NoError(t, err)
	})

	t.Run("retrieve first height", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &mocks.GenericHeight, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got uint64
		err = db.View(l.RetrieveFirst(&got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_SaveAndRetrieveLast(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testKey := encodeKey(prefixLast)

	t.Run("save last height", func(t *testing.T) {

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, uint64(0), v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveLast(mocks.GenericHeight))
		assert.NoError(t, err)
	})

	t.Run("retrieve last height", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &mocks.GenericHeight, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got uint64
		err = db.View(l.RetrieveLast(&got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_SaveAndRetrieveCommit(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testKey := encodeKey(prefixCommit, mocks.GenericHeight)

	t.Run("save commit", func(t *testing.T) {

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, flow.StateCommitment{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveCommit(mocks.GenericHeight, mocks.GenericCommit(0)))
		assert.NoError(t, err)
	})

	t.Run("retrieve commit", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &flow.StateCommitment{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got flow.StateCommitment
		err = db.View(l.RetrieveCommit(mocks.GenericHeight, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_SaveAndRetrieveHeader(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testKey := encodeKey(prefixHeader, mocks.GenericHeight)

	t.Run("save header", func(t *testing.T) {

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, &flow.Header{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveHeader(mocks.GenericHeight, mocks.GenericHeader))

		assert.NoError(t, err)
	})

	t.Run("retrieve header", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &flow.Header{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got flow.Header
		err = db.View(l.RetrieveHeader(mocks.GenericHeight, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_SaveAndRetrieveEvents(t *testing.T) {
	testKey1 := encodeKey(prefixEvents, mocks.GenericHeight, xxhash.ChecksumString64(string(mocks.GenericEventType(0))))
	testKey2 := encodeKey(prefixEvents, mocks.GenericHeight, xxhash.ChecksumString64(string(mocks.GenericEventType(1))))

	t.Run("save multiple events under different types", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		testEvents1 := []flow.Event{
			{Type: mocks.GenericEventType(0)},
			{Type: mocks.GenericEventType(0)},
			{Type: mocks.GenericEventType(0)},
		}
		testEvents2 := []flow.Event{
			{Type: mocks.GenericEventType(1)},
			{Type: mocks.GenericEventType(1)},
			{Type: mocks.GenericEventType(1)},
		}

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, []flow.Event{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveEvents(mocks.GenericHeight, mocks.GenericEventType(0), testEvents1))
		assert.NoError(t, err)

		err = db.Update(l.SaveEvents(mocks.GenericHeight, mocks.GenericEventType(1), testEvents2))
		assert.NoError(t, err)
	})

	t.Run("retrieve events nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, mocks.GenericByteSlice)
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericByteSlice)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &[]flow.Event{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got []flow.Event
		err = db.View(l.RetrieveEvents(mocks.GenericHeight, mocks.GenericEventTypes(2), &got))

		assert.NoError(t, err)
		assert.Equal(t, 2, decodeCallCount)
	})

	t.Run("retrieve events returns all types when no filter given", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, mocks.GenericByteSlice)
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericByteSlice)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &[]flow.Event{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got []flow.Event
		err = db.View(l.RetrieveEvents(mocks.GenericHeight, []flow.EventType{}, &got))

		assert.Equal(t, 2, decodeCallCount)
		assert.NoError(t, err)
	})

	t.Run("retrieve events does not include types not asked for", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, []byte(`value1`))
			require.NoError(t, err)

			err = tx.Set(testKey2, []byte(`value2`))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, []byte(`value1`), b)
			assert.IsType(t, &[]flow.Event{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got []flow.Event
		err = db.View(l.RetrieveEvents(mocks.GenericHeight, []flow.EventType{mocks.GenericEventType(0), "another-type"}, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})

	t.Run("retrieve events does not include types not asked for", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, []byte(`value1`))
			require.NoError(t, err)

			err = tx.Set(testKey2, []byte(`value2`))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, []byte(`value2`), b)
			assert.IsType(t, &[]flow.Event{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got []flow.Event
		err = db.View(l.RetrieveEvents(mocks.GenericHeight, []flow.EventType{"another-type", mocks.GenericEventType(1)}, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_SaveAndRetrievePayload(t *testing.T) {
	testKey1 := encodeKey(prefixPayload, mocks.GenericLedgerPath(0), mocks.GenericHeight)
	testKey2 := encodeKey(prefixPayload, mocks.GenericLedgerPath(0), mocks.GenericHeight*2)

	t.Run("save two different payloads for same path at different heights", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, &ledger.Payload{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SavePayload(mocks.GenericHeight, mocks.GenericLedgerPath(0), mocks.GenericLedgerPayload(0)))
		assert.NoError(t, err)

		err = db.Update(l.SavePayload(mocks.GenericHeight*2, mocks.GenericLedgerPath(0), mocks.GenericLedgerPayload(1)))
		assert.NoError(t, err)
	})

	t.Run("save and retrieve payload at its first indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, mocks.GenericLedgerValue(0))
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericLedgerValue(1))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			// We should find the first value since it's the indexed value at GenericHeight.
			assert.Equal(t, []byte(mocks.GenericLedgerValue(0)), b)
			assert.IsType(t, &ledger.Payload{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(mocks.GenericHeight, mocks.GenericLedgerPath(0), &got))

		assert.NoError(t, err)
	})

	t.Run("retrieve payload at its second indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, mocks.GenericLedgerValue(0))
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericLedgerValue(1))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			// We should find the second value since it's the indexed value at GenericHeight*2.
			assert.Equal(t, []byte(mocks.GenericLedgerValue(1)), b)
			assert.IsType(t, &ledger.Payload{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(mocks.GenericHeight*2, mocks.GenericLedgerPath(0), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})

	t.Run("retrieve payload between first and second indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, mocks.GenericLedgerValue(0))
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericLedgerValue(1))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			// We should find the first value since it's the last indexed value at any
			// height between GenericHeight and GenericHeight*2.
			assert.Equal(t, []byte(mocks.GenericLedgerValue(0)), b)
			assert.IsType(t, &ledger.Payload{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(mocks.GenericHeight+mocks.GenericHeight/2, mocks.GenericLedgerPath(0), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})

	t.Run("retrieve payload after last indexed", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, mocks.GenericLedgerValue(0))
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericLedgerValue(1))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			// We should find the second value since it's the last indexed value
			// at any height beyond the last indexed one.
			assert.Equal(t, []byte(mocks.GenericLedgerValue(1)), b)
			assert.IsType(t, &ledger.Payload{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(999*mocks.GenericHeight, mocks.GenericLedgerPath(0), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})

	t.Run("retrieve payload before it was ever indexed", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, mocks.GenericLedgerValue(0))
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericLedgerValue(1))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(mocks.GenericHeight/2, mocks.GenericLedgerPath(0), &got))

		assert.Error(t, err)
		assert.Equal(t, 0, decodeCallCount) // Should never be called since key does not match anything.
	})

	t.Run("should fail if path does not match", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, mocks.GenericLedgerValue(0))
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericLedgerValue(1))
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		unknownPath := ledger.Path{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(mocks.GenericHeight, unknownPath, &got))

		assert.Error(t, err)
		assert.Equal(t, 0, decodeCallCount) // Should never be called since key does not match anything.
	})
}

func TestLibrary_IndexAndLookupHeightForBlock(t *testing.T) {
	testKey := encodeKey(prefixHeightForBlock, mocks.GenericIdentifier(0))

	t.Run("save height of block", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, uint64(0), v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.IndexHeightForBlock(mocks.GenericIdentifier(0), mocks.GenericHeight))

		assert.NoError(t, err)
	})

	t.Run("retrieve height of block", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &mocks.GenericHeight, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got uint64
		err = db.View(l.LookupHeightForBlock(mocks.GenericIdentifier(0), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestSaveAndRetrieve_Transaction(t *testing.T) {
	testKey := encodeKey(prefixTransaction, mocks.GenericIdentifier(0))

	t.Run("save transaction", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, &flow.TransactionBody{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveTransaction(mocks.GenericTransaction(0)))

		assert.NoError(t, err)
	})

	t.Run("retrieve transaction", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &flow.TransactionBody{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got flow.TransactionBody
		err = db.View(l.RetrieveTransaction(mocks.GenericIdentifier(0), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestIndexAndLookup_TransactionsForHeight(t *testing.T) {
	testKey := encodeKey(prefixTransactionsForHeight, mocks.GenericHeight)

	t.Run("save transactions", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, []flow.Identifier{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.IndexTransactionsForHeight(mocks.GenericHeight, mocks.GenericIdentifiers(5)))

		assert.NoError(t, err)
	})

	t.Run("retrieve transactions", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericLedgerValue(0))
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, []byte(mocks.GenericLedgerValue(0)), b)
			assert.IsType(t, &[]flow.Identifier{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got []flow.Identifier
		err = db.View(l.LookupTransactionsForHeight(mocks.GenericHeight, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestSaveAndRetrieve_Collection(t *testing.T) {
	testKey := encodeKey(prefixCollection, mocks.GenericIdentifier(0))

	t.Run("save collection", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, &flow.LightCollection{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveCollection(mocks.GenericCollection(0)))

		assert.NoError(t, err)
	})

	t.Run("retrieve collection", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &flow.LightCollection{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got flow.LightCollection
		err = db.View(l.RetrieveCollection(mocks.GenericIdentifier(0), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestIndexAndLookup_CollectionsForHeight(t *testing.T) {
	testKey := encodeKey(prefixCollectionsForHeight, mocks.GenericHeight)

	t.Run("save collections", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, []flow.Identifier{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.IndexCollectionsForHeight(mocks.GenericHeight, mocks.GenericIdentifiers(5)))

		assert.NoError(t, err)
	})

	t.Run("retrieve collections", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &[]flow.Identifier{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got []flow.Identifier
		err = db.View(l.LookupCollectionsForHeight(mocks.GenericHeight, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestSaveAndRetrieve_TransactionResults(t *testing.T) {
	testKey := encodeKey(prefixTransactionResults, mocks.GenericResult(0).TransactionID)

	t.Run("save transaction result", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, &flow.TransactionResult{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveTransactionResult(mocks.GenericResult(0)))

		assert.NoError(t, err)
	})

	t.Run("retrieve transaction result", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericByteSlice)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericByteSlice, b)
			assert.IsType(t, &flow.TransactionResult{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got *flow.TransactionResult
		err = db.View(l.RetrieveTransactionResult(mocks.GenericResult(0).TransactionID, got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}
