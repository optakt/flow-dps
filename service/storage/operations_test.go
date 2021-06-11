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

package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/testing/helpers"
)

func TestSaveAndRetrieve_First(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	t.Run("save first height", func(t *testing.T) {
		txn := db.NewTransaction(true)
		err := SaveFirst(42)(txn)

		assert.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)
	})

	t.Run("retrieve first height", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got uint64
		err := RetrieveFirst(&got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, uint64(42), got)
	})
}

func TestSaveAndRetrieve_Last(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	t.Run("save last height", func(t *testing.T) {
		txn := db.NewTransaction(true)

		err := SaveLast(42)(txn)

		assert.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)
	})

	t.Run("retrieve last height", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got uint64
		err := RetrieveLast(&got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, uint64(42), got)
	})
}

func TestSaveAndRetrieve_Commit(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	commit, _ := flow.ToStateCommitment([]byte("07018030187ecf04945f35f1e33a89dc"))

	t.Run("save commit", func(t *testing.T) {
		txn := db.NewTransaction(true)

		err := SaveCommit(42, commit)(txn)

		assert.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)
	})

	t.Run("retrieve commit", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got flow.StateCommitment
		err := RetrieveCommit(42, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, commit, got)
	})
}

func TestSaveAndRetrieve_Header(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	header := &flow.Header{ChainID: "flow-testnet"}

	t.Run("save header", func(t *testing.T) {
		txn := db.NewTransaction(true)

		err := SaveHeader(42, header)(txn)

		assert.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)
	})

	t.Run("retrieve header", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got flow.Header
		err := RetrieveHeader(42, &got)(txn)

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
		txn := db.NewTransaction(true)

		err := SaveEvents(42, testTyp1, testEvents1)(txn)

		assert.NoError(t, err)

		err = SaveEvents(42, testTyp2, testEvents2)(txn)

		assert.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)
	})

	t.Run("retrieve events nominal case", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got []flow.Event
		err := RetrieveEvents(42, []flow.EventType{testTyp1, testTyp2}, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, append(testEvents1, testEvents2...), got)
	})

	t.Run("retrieve events returns all types when no filter given", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got []flow.Event
		err := RetrieveEvents(42, []flow.EventType{}, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, append(testEvents1, testEvents2...), got)
	})

	t.Run("retrieve events does not include types not asked for", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got []flow.Event
		err := RetrieveEvents(42, []flow.EventType{testTyp1, "another-type"}, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, testEvents1, got)
	})

	t.Run("retrieve events does not include types not asked for", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got []flow.Event
		err := RetrieveEvents(42, []flow.EventType{testTyp2, "another-type"}, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, testEvents2, got)
	})
}

func TestSaveAndRetrieve_Payload(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	path, _ := ledger.ToPath([]byte("aac513eb1a0457700ac3fa8d292513e1"))
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
		txn := db.NewTransaction(true)

		err := SavePayload(42, path, payload1)(txn)

		assert.NoError(t, err)

		err = SavePayload(84, path, payload2)(txn)

		assert.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)
	})

	t.Run("retrieve payload at its first indexed height", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got ledger.Payload
		err := RetrievePayload(42, path, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, *payload1, got)
	})

	t.Run("retrieve payload at its second indexed height", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got ledger.Payload
		err := RetrievePayload(84, path, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, *payload2, got)
	})

	t.Run("retrieve payload between first and second indexed height", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got ledger.Payload
		err := RetrievePayload(63, path, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, *payload1, got)
	})

	t.Run("retrieve payload after last indexed", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got ledger.Payload
		err := RetrievePayload(999, path, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, *payload2, got)
	})

	t.Run("retrieve payload before it was ever indexed", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got ledger.Payload
		err := RetrievePayload(10, path, &got)(txn)

		assert.Error(t, err)
	})

	t.Run("should fail if path does not match", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got ledger.Payload
		err := RetrievePayload(42, ledger.Path{}, &got)(txn)

		assert.Error(t, err)
	})
}

func TestSaveAndRetrieve_Height(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()

	blockID, _ := flow.HexStringToIdentifier("aac513eb1a0457700ac3fa8d292513e18ad7fd70065146b35ab48fa5a6cab007")

	t.Run("save height of block", func(t *testing.T) {
		txn := db.NewTransaction(true)

		err := SaveHeight(blockID, 42)(txn)

		assert.NoError(t, err)

		err = txn.Commit()
		require.NoError(t, err)
	})

	t.Run("retrieve height of block", func(t *testing.T) {
		txn := db.NewTransaction(false)

		var got uint64
		err := RetrieveHeight(blockID, &got)(txn)

		assert.NoError(t, err)
		assert.Equal(t, uint64(42), got)
	})
}
