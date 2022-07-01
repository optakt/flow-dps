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
	"encoding/binary"
	"errors"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/testing/helpers"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestLibrary_Retrieve(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	// Insert test value.
	const testValue = uint64(42)
	testKey := []byte{42}

	t.Run("nominal case", func(t *testing.T) {
		wantEncodedValue := []byte{0x28, 0xb5, 0x2f, 0xfd, 0x7, 0x0, 0x7, 0x81, 0x4a, 0x29, 0x11, 0x0, 0x0, 0x18, 0x2a, 0xc5, 0xb, 0xd5, 0x9d}

		err := insertKeyValue(t, db, testKey, testValue)
		require.NoError(t, err)

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, wantEncodedValue, b)

			ptr, ok := v.(*uint64)
			require.True(t, ok)

			*ptr = 42
			return nil
		}

		l := &Library{
			codec: codec,
		}

		var got uint64
		err = db.View(l.retrieve(testKey, &got))

		require.NoError(t, err)
		assert.Equal(t, testValue, got)
	})

	t.Run("unknown key, should fail", func(t *testing.T) {
		l := &Library{
			codec: mocks.BaselineCodec(t),
		}

		var got uint64
		err := db.View(l.retrieve([]byte{13, 37}, &got))

		require.Error(t, err)
		assert.True(t, errors.Is(err, badger.ErrKeyNotFound))

	})

	t.Run("badly encoded value, should fail", func(t *testing.T) {
		wantUnencodedValue := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2a}

		err := insertUnencodedKeyValue(t, db, testKey, testValue)
		require.NoError(t, err)

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(b []byte, v interface{}) error {
			assert.Equal(t, wantUnencodedValue, b)
			return mocks.GenericError
		}

		l := &Library{
			codec: codec,
		}

		var got uint64
		err = db.View(l.retrieve(testKey, &got))

		assert.Error(t, err)
	})
}

func TestLibrary_Save(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	t.Run("nominal case", func(t *testing.T) {

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.IsType(t, uint64(0), v)
			return []byte{}, nil
		}
		l := &Library{
			codec: codec,
		}

		err := db.Update(l.save([]byte{13, 37}, uint64(42)))

		assert.NoError(t, err)
	})

	t.Run("saving a nil value should work", func(t *testing.T) {

		codec := mocks.BaselineCodec(t)
		codec.MarshalFunc = func(v interface{}) ([]byte, error) {
			assert.Nil(t, v)
			return []byte{}, nil
		}

		l := &Library{
			codec: codec,
		}

		err := db.Update(l.save([]byte{13, 37}, nil))

		assert.NoError(t, err)
	})

	t.Run("saving a value at an empty key should fail", func(t *testing.T) {
		l := &Library{
			codec: mocks.BaselineCodec(t),
		}

		err := db.Update(l.save([]byte{}, uint64(42)))

		assert.Error(t, err)
	})
}

func insertKeyValue(t *testing.T, db *badger.DB, key []byte, value uint64) error {
	t.Helper()

	err := db.Update(func(txn *badger.Txn) error {
		enc := zbor.NewCodec()

		val, err := enc.Marshal(value)
		require.NoError(t, err)

		return txn.Set(key, val)
	})
	return err
}

func insertUnencodedKeyValue(t *testing.T, db *badger.DB, key []byte, value uint64) error {
	t.Helper()

	err := db.Update(func(txn *badger.Txn) error {
		val := make([]byte, 8)
		binary.BigEndian.PutUint64(val, value)

		return txn.Set(key, val)
	})
	return err
}
