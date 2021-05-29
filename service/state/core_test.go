// Copyright 2021 Alvalor S.A.
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

package state

import (
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/service/storage"
)

var (
	lastHeight = uint64(42)
	lastCommit = flow.StateCommitment{132, 131, 130, 129, 128, 127, 126, 125, 124, 123, 122, 121, 120, 119, 118, 117, 116, 115, 114, 113, 112, 111, 110, 19, 18, 17, 16, 15, 14, 13, 12, 11}

	testBlockID   = flow.Identifier{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	testEventType = flow.EventType("testType")
	testEvents    = []flow.Event{{Type: testEventType}, {Type: testEventType}, {Type: testEventType}}
	testHeader    = &flow.Header{ChainID: "test-chain"}
	testKeyParts  = []ledger.KeyPart{{Type: 0, Value: []byte(`testOwner`)}, {Type: 1, Value: []byte(`testController`)}, {Type: 2, Value: []byte(`testKey`)}}
	testKey       = ledger.Key{KeyParts: testKeyParts}
	testKeys      = []ledger.Key{testKey, testKey, testKey}
	testValue     = []byte(`testValue`)
	testValues    = []ledger.Value{testValue, testValue, testValue}
	testPayload   = &ledger.Payload{Key: testKey, Value: testValue}
	testKeyHex    = []byte{139, 55, 7, 236, 49, 129, 213, 248, 52, 211, 245, 24, 153, 110, 242, 149, 113, 114, 147, 169, 240, 99, 17, 107, 174, 82, 185, 178, 170, 228, 221, 138}
	testPath, _   = ledger.ToPath(testKeyHex)
)

func TestCore_Ledger(t *testing.T) {
	c := &Core{}
	l := c.Ledger()

	led, ok := l.(*Ledger)
	assert.True(t, ok)

	assert.Equal(t, c, led.core)
	assert.Equal(t, uint8(complete.DefaultPathFinderVersion), led.version)
}

func TestCore_payload(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{
		db:     db,
		height: lastHeight,
		commit: lastCommit,
	}

	t.Run("should return proper payload on known key path", func(t *testing.T) {
		payload, err := c.payload(lastHeight, testPath)
		assert.NoError(t, err)
		assert.Equal(t, testPayload, payload)
	})

	t.Run("should return empty payload on unknown key path", func(t *testing.T) {
		payload, err := c.payload(lastHeight, ledger.Path{})
		assert.NoError(t, err)
		assert.Equal(t, ledger.EmptyPayload(), payload)
	})

	t.Run("should error when given a height above last indexed height", func(t *testing.T) {
		p, err := c.payload(2*lastHeight, ledger.Path{})
		assert.Error(t, err)
		assert.Nil(t, p)
	})
}

func inMemoryDB(t *testing.T) *badger.DB {
	t.Helper()

	opts := badger.DefaultOptions("")
	opts.InMemory = true
	opts.Logger = nil

	db, err := badger.Open(opts)
	require.NoError(t, err)

	err = db.Update(func(txn *badger.Txn) error {
		err = storage.SavePayload(lastHeight, testPath, testPayload)(txn)
		if err != nil {
			return err
		}

		err = storage.SaveCommitForHeight(lastCommit, lastHeight)(txn)
		if err != nil {
			return err
		}

		err = storage.SaveHeightForCommit(lastHeight, lastCommit)(txn)
		if err != nil {
			return err
		}

		err = storage.SaveHeaderForHeight(lastHeight, testHeader)(txn)
		if err != nil {
			return err
		}

		err = storage.SaveEvents(lastHeight, testEventType, testEvents)(txn)
		if err != nil {
			return err
		}

		err = storage.SaveHeightForBlock(testBlockID, lastHeight-1)(txn)
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(t, err)

	return db
}
