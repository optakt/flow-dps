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

	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/service/loader"
	"github.com/onflow/flow-dps/testing/helpers"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestLibrary_SaveAndRetrieveFirst(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testKey := EncodeKey(PrefixFirst)

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
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
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

	testKey := EncodeKey(PrefixLast)

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
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
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

	testKey := EncodeKey(PrefixCommit, mocks.GenericHeight)

	t.Run("save commit", func(t *testing.T) {

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, flow.DummyStateCommitment, v)
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
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
			assert.IsType(t, &flow.DummyStateCommitment, v)
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

	testKey := EncodeKey(PrefixHeader, mocks.GenericHeight)

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
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
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
	testKey1 := EncodeKey(PrefixEvents, mocks.GenericHeight, xxhash.ChecksumString64(string(mocks.GenericEventType(0))))
	testKey2 := EncodeKey(PrefixEvents, mocks.GenericHeight, xxhash.ChecksumString64(string(mocks.GenericEventType(1))))

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
			err := tx.Set(testKey1, mocks.GenericBytes)
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericBytes)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
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
			err := tx.Set(testKey1, mocks.GenericBytes)
			require.NoError(t, err)

			err = tx.Set(testKey2, mocks.GenericBytes)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
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
	testKey1 := EncodeKey(PrefixPayload, mocks.GenericLedgerPath(0), mocks.GenericHeight)
	testKey2 := EncodeKey(PrefixPayload, mocks.GenericLedgerPath(0), mocks.GenericHeight*2)

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
	blockID := mocks.GenericHeader.ID()
	testKey := EncodeKey(PrefixHeightForBlock, blockID)

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

		err := db.Update(l.IndexHeightForBlock(blockID, mocks.GenericHeight))

		assert.NoError(t, err)
	})

	t.Run("retrieve height of block", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
			assert.IsType(t, &mocks.GenericHeight, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got uint64
		err = db.View(l.LookupHeightForBlock(blockID, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestSaveAndRetrieve_Transaction(t *testing.T) {
	tx := mocks.GenericTransaction(0)
	testKey := EncodeKey(PrefixTransaction, tx.ID())

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

		err := db.Update(l.SaveTransaction(tx))

		assert.NoError(t, err)
	})

	t.Run("retrieve transaction", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
			assert.IsType(t, &flow.TransactionBody{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got flow.TransactionBody
		err = db.View(l.RetrieveTransaction(tx.ID(), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_IndexAndLookupHeightForTransaction(t *testing.T) {
	txID := mocks.GenericHeader.ID()
	testKey := EncodeKey(PrefixHeightForTransaction, txID)

	t.Run("save height of transaction", func(t *testing.T) {
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

		err := db.Update(l.IndexHeightForTransaction(txID, mocks.GenericHeight))

		assert.NoError(t, err)
	})

	t.Run("retrieve height of transaction", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
			assert.IsType(t, &mocks.GenericHeight, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got uint64
		err = db.View(l.LookupHeightForTransaction(txID, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestIndexAndLookup_TransactionsForHeight(t *testing.T) {
	testKey := EncodeKey(PrefixTransactionsForHeight, mocks.GenericHeight)

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

		err := db.Update(l.IndexTransactionsForHeight(mocks.GenericHeight, mocks.GenericTransactionIDs(5)))

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
	collection := mocks.GenericCollection(0)
	testKey := EncodeKey(PrefixCollection, collection.ID())

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

		err := db.Update(l.SaveCollection(collection))

		assert.NoError(t, err)
	})

	t.Run("retrieve collection", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
			assert.IsType(t, &flow.LightCollection{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got flow.LightCollection
		err = db.View(l.RetrieveCollection(collection.ID(), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestSaveAndRetrieve_Guarantee(t *testing.T) {
	guarantee := mocks.GenericGuarantee(0)
	testKey := EncodeKey(PrefixGuarantee, guarantee.ID())

	t.Run("save guarantee", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, &flow.CollectionGuarantee{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveGuarantee(guarantee))

		assert.NoError(t, err)
	})

	t.Run("retrieve guarantee", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
			assert.IsType(t, &flow.CollectionGuarantee{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got flow.CollectionGuarantee
		err = db.View(l.RetrieveGuarantee(guarantee.ID(), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestIndexAndLookup_CollectionsForHeight(t *testing.T) {
	collIDs := mocks.GenericCollectionIDs(5)
	testKey := EncodeKey(PrefixCollectionsForHeight, mocks.GenericHeight)

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

		err := db.Update(l.IndexCollectionsForHeight(mocks.GenericHeight, collIDs))

		assert.NoError(t, err)
	})

	t.Run("retrieve collections", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
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
	testKey := EncodeKey(PrefixResults, mocks.GenericResult(0).TransactionID)

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

		err := db.Update(l.SaveResult(mocks.GenericResult(0)))

		assert.NoError(t, err)
	})

	t.Run("retrieve transaction result", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
			assert.IsType(t, &flow.TransactionResult{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got *flow.TransactionResult
		err = db.View(l.RetrieveResult(mocks.GenericResult(0).TransactionID, got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestSaveAndRetrieve_Seal(t *testing.T) {
	seal := mocks.GenericSeal(0)
	testKey := EncodeKey(PrefixSeal, seal.ID())

	t.Run("save seal", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, &flow.Seal{}, v)
			return mocks.GenericLedgerValue(0), nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.SaveSeal(seal))

		assert.NoError(t, err)
	})

	t.Run("retrieve seal", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, mocks.GenericBytes)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, mocks.GenericBytes, b)
			assert.IsType(t, &flow.Seal{}, v)
			decodeCallCount++

			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got flow.Seal
		err = db.View(l.RetrieveSeal(seal.ID(), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestIndexAndLookup_Seals(t *testing.T) {
	testKey := EncodeKey(PrefixSealsForHeight, mocks.GenericHeight)

	t.Run("save seals", func(t *testing.T) {
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

		err := db.Update(l.IndexSealsForHeight(mocks.GenericHeight, mocks.GenericSealIDs(5)))
		assert.NoError(t, err)
	})

	t.Run("retrieve seals", func(t *testing.T) {
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
		err = db.View(l.LookupSealsForHeight(mocks.GenericHeight, &got))
		require.NoError(t, err)

		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_IterateLedger(t *testing.T) {
	entries := 5
	paths := mocks.GenericLedgerPaths(entries)
	payloads := mocks.GenericLedgerPayloads(entries)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := zbor.NewCodec()
		l := &Library{codec}

		for i := 0; i < entries; i++ {
			height := mocks.GenericHeight + uint64(i)
			require.NoError(t, db.Update(l.SavePayload(height, paths[i], payloads[i])))
		}

		got := make(map[ledger.Path]*ledger.Payload)
		op := l.IterateLedger(loader.ExcludeNone(), func(path ledger.Path, payload *ledger.Payload) error {
			got[path] = payload

			return nil
		})

		err := db.View(op)

		assert.NoError(t, err)
		assert.Len(t, got, entries)
		for i := 0; i < entries; i++ {
			assert.Equal(t, payloads[i], got[paths[i]])
		}
	})

	t.Run("handles multiple payloads with the same path", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := zbor.NewCodec()
		l := &Library{codec}

		// Always use paths[0] for every payload.
		path := paths[0]
		for i := 0; i < entries; i++ {
			height := mocks.GenericHeight + uint64(i)
			require.NoError(t, db.Update(l.SavePayload(height, path, payloads[i])))
		}

		got := make(map[ledger.Path]*ledger.Payload)
		op := l.IterateLedger(loader.ExcludeNone(), func(path ledger.Path, payload *ledger.Payload) error {
			got[path] = payload

			return nil
		})

		err := db.View(op)

		require.NoError(t, err)
		// Verify that only the payload with the greatest height was used in the callback.
		assert.Len(t, got, 1)
		assert.Equal(t, payloads[entries-1], got[path])
	})

	t.Run("handles codec failure", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func([]byte, interface{}) error {
			return mocks.GenericError
		}
		l := &Library{codec}

		for i := 0; i < entries; i++ {
			height := mocks.GenericHeight + uint64(i)
			require.NoError(t, db.Update(l.SavePayload(height, paths[i], payloads[i])))
		}

		got := make(map[ledger.Path]*ledger.Payload)
		op := l.IterateLedger(loader.ExcludeNone(), func(path ledger.Path, payload *ledger.Payload) error {
			got[path] = payload

			return nil
		})

		err := db.View(op)

		assert.Error(t, err)
		assert.Len(t, got, 0)
	})

	t.Run("handles callback error", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		codec := zbor.NewCodec()
		l := &Library{codec}

		for i := 0; i < entries; i++ {
			height := mocks.GenericHeight + uint64(i)
			require.NoError(t, db.Update(l.SavePayload(height, paths[i], payloads[i])))
		}

		op := l.IterateLedger(loader.ExcludeNone(), func(path ledger.Path, payload *ledger.Payload) error {
			return mocks.GenericError
		})

		err := db.View(op)

		assert.Error(t, err)
	})
}
