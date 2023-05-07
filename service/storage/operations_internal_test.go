package storage

import (
	"testing"

	"github.com/OneOfOne/xxhash"
	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-archive/testing/helpers"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestLibrary_SaveAndRetrieveFirst(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testKey := EncodeKey(PrefixFirst)

	t.Run("save first height", func(t *testing.T) {

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, uint64(0), v)
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return tx.Set(testKey, mocks.GenericRegisterValue(0))
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, []byte(mocks.GenericRegisterValue(0)), b)
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return mocks.GenericRegisterValue(0), nil
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
			return tx.Set(testKey, mocks.GenericRegisterValue(0))
		})
		require.NoError(t, err)

		decodeCallCount := 0
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, []byte(mocks.GenericRegisterValue(0)), b)
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
