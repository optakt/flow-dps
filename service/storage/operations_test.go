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

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/testing/helpers"
)

func TestSaveAndRetrieve_First(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	t.Run("save first height", func(t *testing.T) {
		err := db.Update(SaveFirst(42))
		assert.NoError(t, err)
	})

	t.Run("retrieve first height", func(t *testing.T) {
		var got uint64
		err := db.View(RetrieveFirst(&got))

		assert.NoError(t, err)
		assert.Equal(t, uint64(42), got)
	})
}

func TestSaveAndRetrieve_Last(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	t.Run("save last height", func(t *testing.T) {
		err := db.Update(SaveLast(42))
		assert.NoError(t, err)
	})

	t.Run("retrieve last height", func(t *testing.T) {
		var got uint64
		err := db.View(RetrieveLast(&got))

		assert.NoError(t, err)
		assert.Equal(t, uint64(42), got)
	})
}

func TestSaveAndRetrieve_Commit(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	commit, _ := flow.ToStateCommitment([]byte("07018030187ecf04945f35f1e33a89dc"))

	t.Run("save commit", func(t *testing.T) {
		err := db.Update(SaveCommit(42, commit))
		assert.NoError(t, err)
	})

	t.Run("retrieve commit", func(t *testing.T) {
		var got flow.StateCommitment
		err := db.View(RetrieveCommit(42, &got))

		assert.NoError(t, err)
		assert.Equal(t, commit, got)
	})
}

func TestSaveAndRetrieve_Header(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	header := &flow.Header{ChainID: "flow-testnet"}

	t.Run("save header", func(t *testing.T) {
		err := db.Update(SaveHeader(42, header))

		assert.NoError(t, err)
	})

	t.Run("retrieve header", func(t *testing.T) {
		var got flow.Header
		err := db.View(RetrieveHeader(42, &got))

		assert.NoError(t, err)
		assert.Equal(t, *header, got)
	})
}

func TestSaveAndRetrieve_Events(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testTyp1 := flow.EventType("test1")
	testEvents1 := []flow.Event{{Type: testTyp1}, {Type: testTyp1}, {Type: testTyp1}}
	testTyp2 := flow.EventType("test2")
	testEvents2 := []flow.Event{{Type: testTyp2}, {Type: testTyp2}, {Type: testTyp2}}

	t.Run("save multiple events under different types", func(t *testing.T) {
		err := db.Update(SaveEvents(42, testTyp1, testEvents1))
		assert.NoError(t, err)

		err = db.Update(SaveEvents(42, testTyp2, testEvents2))
		assert.NoError(t, err)
	})

	t.Run("retrieve events nominal case", func(t *testing.T) {
		var got []flow.Event
		err := db.Update(RetrieveEvents(42, []flow.EventType{testTyp1, testTyp2}, &got))

		assert.NoError(t, err)
		assert.Equal(t, append(testEvents1, testEvents2...), got)
	})

	t.Run("retrieve events returns all types when no filter given", func(t *testing.T) {
		var got []flow.Event
		err := db.Update(RetrieveEvents(42, []flow.EventType{}, &got))

		assert.NoError(t, err)
		assert.Equal(t, append(testEvents1, testEvents2...), got)
	})

	t.Run("retrieve events does not include types not asked for", func(t *testing.T) {
		var got []flow.Event
		err := db.Update(RetrieveEvents(42, []flow.EventType{testTyp1, "another-type"}, &got))

		assert.NoError(t, err)
		assert.Equal(t, testEvents1, got)
	})

	t.Run("retrieve events does not include types not asked for", func(t *testing.T) {
		var got []flow.Event
		err := db.Update(RetrieveEvents(42, []flow.EventType{testTyp2, "another-type"}, &got))

		assert.NoError(t, err)
		assert.Equal(t, testEvents2, got)
	})
}

func TestSaveAndRetrieve_Payload(t *testing.T) {
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

	t.Run("save two different payloads for same path at different heights", func(t *testing.T) {
		err := db.Update(SavePayload(42, path, payload1))
		assert.NoError(t, err)

		err = db.Update(SavePayload(84, path, payload2))
		assert.NoError(t, err)
	})

	t.Run("retrieve payload at its first indexed height", func(t *testing.T) {
		var got ledger.Payload
		err := db.View(RetrievePayload(42, path, &got))

		assert.NoError(t, err)
		assert.Equal(t, *payload1, got)
	})

	t.Run("retrieve payload at its second indexed height", func(t *testing.T) {
		var got ledger.Payload
		err := db.View(RetrievePayload(84, path, &got))

		assert.NoError(t, err)
		assert.Equal(t, *payload2, got)
	})

	t.Run("retrieve payload between first and second indexed height", func(t *testing.T) {
		var got ledger.Payload
		err := db.View(RetrievePayload(63, path, &got))

		assert.NoError(t, err)
		assert.Equal(t, *payload1, got)
	})

	t.Run("retrieve payload after last indexed", func(t *testing.T) {
		var got ledger.Payload
		err := db.View(RetrievePayload(999, path, &got))

		assert.NoError(t, err)
		assert.Equal(t, *payload2, got)
	})

	t.Run("retrieve payload before it was ever indexed", func(t *testing.T) {
		var got ledger.Payload
		err := db.View(RetrievePayload(10, path, &got))

		assert.Error(t, err)
	})

	t.Run("should fail if path does not match", func(t *testing.T) {
		var got ledger.Payload
		unknownPath := ledger.Path{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
		err := db.View(RetrievePayload(42, unknownPath, &got))

		assert.Error(t, err)
	})
}

func TestSaveAndRetrieve_Height(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	blockID, _ := flow.HexStringToIdentifier("aac513eb1a0457700ac3fa8d292513e18ad7fd70065146b35ab48fa5a6cab007")

	t.Run("save height of block", func(t *testing.T) {
		err := db.Update(SaveHeight(blockID, 42))

		assert.NoError(t, err)
	})

	t.Run("retrieve height of block", func(t *testing.T) {
		var got uint64
		err := db.View(RetrieveHeight(blockID, &got))

		assert.NoError(t, err)
		assert.Equal(t, uint64(42), got)
	})
}

func TestSaveAndRetrieve_Transaction(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testID := flow.Identifier{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1, 0x8a, 0xd7, 0xfd, 0x70, 0x06, 0x51, 0x46, 0xb3, 0x5a, 0xb4, 0x8f, 0xa5, 0xa6, 0xca, 0xb0, 0x07}
	testTransaction := flow.Transaction{
		TransactionBody: flow.TransactionBody{
			ReferenceBlockID: testID,
		},
	}

	t.Run("save transaction", func(t *testing.T) {
		err := db.Update(SaveTransaction(testTransaction))

		assert.NoError(t, err)
	})

	t.Run("retrieve transaction", func(t *testing.T) {
		var got flow.Transaction
		err := db.View(RetrieveTransaction(testTransaction.ID(), &got))

		assert.NoError(t, err)
		assert.Equal(t, testTransaction, got)
	})
}

func TestSaveAndRetrieve_Transactions(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	testBlockID := flow.Identifier{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1, 0x8a, 0xd7, 0xfd, 0x70, 0x06, 0x51, 0x46, 0xb3, 0x5a, 0xb4, 0x8f, 0xa5, 0xa6, 0xca, 0xb0, 0x07}
	testTransactionID := flow.Identifier{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	testTransactions := []flow.Identifier{testTransactionID, testTransactionID, testTransactionID, testTransactionID, testTransactionID}

	t.Run("save transactions", func(t *testing.T) {
		err := db.Update(SaveTransactions(testBlockID, testTransactions))

		assert.NoError(t, err)
	})

	t.Run("retrieve transactions", func(t *testing.T) {
		var got []flow.Identifier
		err := db.View(RetrieveTransactions(testBlockID, &got))

		assert.NoError(t, err)
		assert.Equal(t, testTransactions, got)
	})
}
