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
	"bytes"
	"testing"

	"github.com/OneOfOne/xxhash"
	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/testing/mocks"

	"github.com/optakt/flow-dps/testing/helpers"
)

func TestLibrary_SaveAndRetrieveFirst(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testHeight := uint64(42)
	testValue := []byte(`testHeight`)
	testKey := encodeKey(prefixFirst)

	t.Run("save first height", func(t *testing.T) {

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, uint64(0), v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.SaveFirst(testHeight))
		assert.NoError(t, err)
	})

	t.Run("retrieve first height", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &testHeight, v)
					decodeCallCount++

					return nil
				},
			},
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

	testHeight := uint64(42)
	testValue := []byte(`testHeight`)
	testKey := encodeKey(prefixLast)

	t.Run("save last height", func(t *testing.T) {

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, uint64(0), v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.SaveLast(testHeight))
		assert.NoError(t, err)
	})

	t.Run("retrieve last height", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &testHeight, v)
					decodeCallCount++

					return nil
				},
			},
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

	commit, _ := flow.ToStateCommitment([]byte("07018030187ecf04945f35f1e33a89dc"))
	testHeight := uint64(42)
	testKey := encodeKey(prefixCommit, testHeight)
	testValue := []byte(`testCommit`)

	t.Run("save commit", func(t *testing.T) {
		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, flow.StateCommitment{}, v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.SaveCommit(42, commit))
		assert.NoError(t, err)
	})

	t.Run("retrieve commit", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &flow.StateCommitment{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got flow.StateCommitment
		err = db.View(l.RetrieveCommit(42, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_SaveAndRetrieveHeader(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	header := flow.Header{ChainID: "flow-testnet"}

	testHeight := uint64(42)
	testKey := encodeKey(prefixHeader, testHeight)
	testValue := []byte(`testHeader`)

	t.Run("save header", func(t *testing.T) {

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, &flow.Header{}, v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.SaveHeader(testHeight, &header))

		assert.NoError(t, err)
	})

	t.Run("retrieve header", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &flow.Header{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got flow.Header
		err = db.View(l.RetrieveHeader(testHeight, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_SaveAndRetrieveEvents(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testHeight := uint64(42)
	testTyp1 := flow.EventType("test1")
	testTyp2 := flow.EventType("test2")
	testEventType1 := xxhash.ChecksumString64(string(testTyp1))
	testEventType2 := xxhash.ChecksumString64(string(testTyp2))

	testKey1 := encodeKey(prefixEvents, testHeight, testEventType1)
	testValue1 := []byte(`events1`)
	testKey2 := encodeKey(prefixEvents, testHeight, testEventType2)
	testValue2 := []byte(`events2`)
	t.Run("save multiple events under different types", func(t *testing.T) {
		testEvents1 := []flow.Event{{Type: testTyp1}, {Type: testTyp1}, {Type: testTyp1}}
		testEvents2 := []flow.Event{{Type: testTyp2}, {Type: testTyp2}, {Type: testTyp2}}

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, []flow.Event{}, v)
					return testValue1, nil
				},
			},
		}

		err := db.Update(l.SaveEvents(42, testTyp1, testEvents1))
		assert.NoError(t, err)

		err = db.Update(l.SaveEvents(42, testTyp2, testEvents2))
		assert.NoError(t, err)
	})

	t.Run("retrieve events nominal case", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, testValue1)
			require.NoError(t, err)

			err = tx.Set(testKey2, testValue2)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					// We should find both batches of events since they are both allowed by filter.
					assert.True(t, bytes.Equal(testValue1, b) || bytes.Equal(testValue2, b))
					assert.IsType(t, &[]flow.Event{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got []flow.Event
		err = db.View(l.RetrieveEvents(42, []flow.EventType{testTyp1, testTyp2}, &got))

		assert.NoError(t, err)
		assert.Equal(t, 2, decodeCallCount)
	})

	t.Run("retrieve events returns all types when no filter given", func(t *testing.T) {
		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					// We should find both batches of events since they are not filtered.
					assert.True(t, bytes.Equal(testValue1, b) || bytes.Equal(testValue2, b))
					assert.IsType(t, &[]flow.Event{}, v)
					decodeCallCount++
					return nil
				},
			},
		}

		var got []flow.Event
		err := db.View(l.RetrieveEvents(42, []flow.EventType{}, &got))

		assert.Equal(t, 2, decodeCallCount)
		assert.NoError(t, err)
	})

	t.Run("retrieve events does not include types not asked for", func(t *testing.T) {
		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					// We should find both only testValue1 because of the filter.
					assert.Equal(t, testValue1, b)
					assert.IsType(t, &[]flow.Event{}, v)
					decodeCallCount++
					return nil
				},
			},
		}

		var got []flow.Event
		err := db.View(l.RetrieveEvents(42, []flow.EventType{testTyp1, "another-type"}, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})

	t.Run("retrieve events does not include types not asked for", func(t *testing.T) {
		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					// We should find both only testValue2 because of the filter.
					assert.Equal(t, testValue2, b)
					assert.IsType(t, &[]flow.Event{}, v)
					decodeCallCount++
					return nil
				},
			},
		}

		var got []flow.Event
		err := db.View(l.RetrieveEvents(42, []flow.EventType{"another-type", testTyp2}, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestLibrary_SaveAndRetrievePayload(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	path := ledger.Path{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1}
	key := ledger.NewKey([]ledger.KeyPart{
		ledger.NewKeyPart(0, []byte(`owner`)),
		ledger.NewKeyPart(1, []byte(`controller`)),
		ledger.NewKeyPart(2, []byte(`key`)),
	})
	payload1 := ledger.NewPayload(
		key,
		ledger.Value(`test1`),
	)
	payload2 := ledger.NewPayload(
		key,
		ledger.Value(`test2`),
	)
	testHeight1 := uint64(42)
	testHeight2 := uint64(84)
	testKey1 := encodeKey(prefixPayload, path, testHeight1)
	testValue1 := []byte(`payload1`)
	testKey2 := encodeKey(prefixPayload, path, testHeight2)
	testValue2 := []byte(`payload2`)

	t.Run("save two different payloads for same path at different heights", func(t *testing.T) {
		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, &ledger.Payload{}, v)
					return testValue1, nil
				},
			},
		}

		err := db.Update(l.SavePayload(42, path, payload1))
		assert.NoError(t, err)

		err = db.Update(l.SavePayload(84, path, payload2))
		assert.NoError(t, err)
	})

	t.Run("save and retrieve payload at its first indexed height", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, testValue1)
			require.NoError(t, err)

			err = tx.Set(testKey2, testValue2)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					// We should find value 1 since it's the indexed value at height 42.
					assert.Equal(t, testValue1, b)
					assert.IsType(t, &ledger.Payload{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(testHeight1, path, &got))

		assert.NoError(t, err)
	})

	t.Run("retrieve payload at its second indexed height", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, testValue1)
			require.NoError(t, err)

			err = tx.Set(testKey2, testValue2)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					// We should find value 1 since it's the indexed value at height 84.
					assert.Equal(t, testValue2, b)
					assert.IsType(t, &ledger.Payload{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(84, path, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})

	t.Run("retrieve payload between first and second indexed height", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, testValue1)
			require.NoError(t, err)

			err = tx.Set(testKey2, testValue2)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					// We should find value 1 since it's the last indexed value at any height between 42 and 84.
					assert.Equal(t, testValue1, b)
					assert.IsType(t, &ledger.Payload{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(63, path, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})

	t.Run("retrieve payload after last indexed", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, testValue1)
			require.NoError(t, err)

			err = tx.Set(testKey2, testValue2)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					// We should find value 2 since it's the last indexed value at height 999.
					assert.Equal(t, testValue2, b)
					assert.IsType(t, &ledger.Payload{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(999, path, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})

	t.Run("retrieve payload before it was ever indexed", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, testValue1)
			require.NoError(t, err)

			err = tx.Set(testKey2, testValue2)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					decodeCallCount++

					return nil
				},
			},
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(10, path, &got))

		assert.Error(t, err)
		assert.Equal(t, 0, decodeCallCount) // Should never be called since key does not match anything.
	})

	t.Run("should fail if path does not match", func(t *testing.T) {
		unknownPath := ledger.Path{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

		err := db.Update(func(tx *badger.Txn) error {
			err := tx.Set(testKey1, testValue1)
			require.NoError(t, err)

			err = tx.Set(testKey2, testValue2)
			require.NoError(t, err)

			return nil
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					decodeCallCount++

					return nil
				},
			},
		}

		var got ledger.Payload
		err = db.View(l.RetrievePayload(42, unknownPath, &got))

		assert.Error(t, err)
		assert.Equal(t, 0, decodeCallCount) // Should never be called since key does not match anything.
	})
}

func TestLibrary_IndexAndLookupHeightForBlock(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testHeight := uint64(42)
	blockID, _ := flow.HexStringToIdentifier("aac513eb1a0457700ac3fa8d292513e18ad7fd70065146b35ab48fa5a6cab007")
	testKey := encodeKey(prefixHeightForBlock, blockID)
	testValue := []byte(`testValue`)

	t.Run("save height of block", func(t *testing.T) {

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, uint64(0), v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.IndexHeightForBlock(blockID, testHeight))

		assert.NoError(t, err)
	})

	t.Run("retrieve height of block", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &testHeight, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got uint64
		err = db.View(l.LookupHeightForBlock(blockID, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestSaveAndRetrieve_Transaction(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testID := flow.Identifier{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1, 0x8a, 0xd7, 0xfd, 0x70, 0x06, 0x51, 0x46, 0xb3, 0x5a, 0xb4, 0x8f, 0xa5, 0xa6, 0xca, 0xb0, 0x07}
	testTransaction := &flow.TransactionBody{
		ReferenceBlockID: testID,
	}
	testKey := encodeKey(prefixTransaction, testID)
	testValue := []byte(`testValue`)

	t.Run("save transaction", func(t *testing.T) {

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, &flow.TransactionBody{}, v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.SaveTransaction(testTransaction))

		assert.NoError(t, err)
	})

	t.Run("retrieve transaction", func(t *testing.T) {

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &flow.TransactionBody{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got flow.TransactionBody
		err = db.View(l.RetrieveTransaction(testTransaction.ID(), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestIndexAndLookup_TransactionsForHeight(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testHeight := uint64(1337)
	testTransactionID := flow.Identifier{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	testTxIDs := []flow.Identifier{testTransactionID, testTransactionID, testTransactionID, testTransactionID, testTransactionID}
	testKey := encodeKey(prefixTransactionsForHeight, testHeight)
	testValue := []byte(`testValue`)

	t.Run("save transactions", func(t *testing.T) {

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, []flow.Identifier{}, v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.IndexTransactionsForHeight(testHeight, testTxIDs))

		assert.NoError(t, err)
	})

	t.Run("retrieve transactions", func(t *testing.T) {

		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &[]flow.Identifier{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got []flow.Identifier
		err = db.View(l.LookupTransactionsForHeight(testHeight, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestSaveAndRetrieve_Collection(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testID := flow.Identifier{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1, 0x8a, 0xd7, 0xfd, 0x70, 0x06, 0x51, 0x46, 0xb3, 0x5a, 0xb4, 0x8f, 0xa5, 0xa6, 0xca, 0xb0, 0x07}
	testCollection := &flow.LightCollection{
		Transactions: []flow.Identifier{testID},
	}
	testKey := encodeKey(prefixCollection, testID)
	testValue := []byte(`testValue`)

	t.Run("save collection", func(t *testing.T) {

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, &flow.LightCollection{}, v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.SaveCollection(testCollection))

		assert.NoError(t, err)
	})

	t.Run("retrieve collection", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &flow.LightCollection{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got flow.LightCollection
		err = db.View(l.RetrieveCollection(testCollection.ID(), &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}

func TestIndexAndLookup_CollectionsForHeight(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testHeight := uint64(1337)
	testCollectionID := flow.Identifier{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	testCollIDs := []flow.Identifier{testCollectionID, testCollectionID, testCollectionID, testCollectionID, testCollectionID}
	testKey := encodeKey(prefixCollectionsForHeight, testHeight)
	testValue := []byte(`testValue`)

	t.Run("save collections", func(t *testing.T) {

		l := &Library{
			codec: &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, []flow.Identifier{}, v)
					return testValue, nil
				},
			},
		}

		err := db.Update(l.IndexCollectionsForHeight(testHeight, testCollIDs))

		assert.NoError(t, err)
	})

	t.Run("retrieve collections", func(t *testing.T) {
		err := db.Update(func(tx *badger.Txn) error {
			return tx.Set(testKey, testValue)
		})
		require.NoError(t, err)

		decodeCallCount := 0
		l := &Library{
			codec: &mocks.Codec{
				UnmarshalFunc: func(b []byte, v interface{}) error {
					assert.Equal(t, testValue, b)
					assert.IsType(t, &[]flow.Identifier{}, v)
					decodeCallCount++

					return nil
				},
			},
		}

		var got []flow.Identifier
		err = db.View(l.LookupCollectionsForHeight(testHeight, &got))

		assert.NoError(t, err)
		assert.Equal(t, 1, decodeCallCount)
	})
}
