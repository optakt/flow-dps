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

package chain_test

import (
	"math"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"
	"github.com/optakt/flow-dps/testing/helpers"

	"github.com/optakt/flow-dps/service/chain"
)

const (
	testHeight  = 42
	testChainID = flow.ChainID("flow-testnet")
)

var (
	testCommit  = flow.StateCommitment{0xa2, 0x04, 0x51, 0x3c, 0xc3, 0xc9, 0xa7, 0xf2, 0xec, 0x08, 0x93, 0x56, 0x5f, 0x52, 0xc2, 0x9e, 0x19, 0xf5, 0x58, 0x88, 0x10, 0x11, 0xe1, 0x13, 0x60, 0x43, 0x9e, 0x57, 0x60, 0x18, 0xe3, 0xde}
	testBlockID = flow.Identifier{0xd5, 0xf5, 0x0b, 0xc1, 0x7b, 0xa1, 0xea, 0xad, 0x83, 0x0c, 0x86, 0xac, 0xce, 0x64, 0x5c, 0xa6, 0xc0, 0x9f, 0xf0, 0xfe, 0xc5, 0x1c, 0x76, 0x10, 0x03, 0x1c, 0xb9, 0x99, 0xa5, 0xb0, 0xb3, 0x22}
)

func TestDisk_Root(t *testing.T) {
	db := populatedDB(t)
	defer db.Close()
	c := chain.FromDisk(db)

	root, err := c.Root()
	assert.NoError(t, err)
	assert.Equal(t, uint64(testHeight), root)
}

func TestDisk_Header(t *testing.T) {
	db := populatedDB(t)
	defer db.Close()
	c := chain.FromDisk(db)

	header, err := c.Header(testHeight)
	assert.NoError(t, err)

	require.NotNil(t, header)
	assert.Equal(t, testChainID, header.ChainID)

	_, err = c.Header(math.MaxUint64)
	assert.Error(t, err)
}

func TestDisk_Commit(t *testing.T) {
	db := populatedDB(t)
	defer db.Close()
	c := chain.FromDisk(db)

	commit, err := c.Commit(testHeight)
	assert.NoError(t, err)
	assert.Equal(t, testCommit, commit)

	_, err = c.Commit(math.MaxUint64)
	assert.Error(t, err)
}

func TestDisk_Events(t *testing.T) {
	db := populatedDB(t)
	defer db.Close()
	c := chain.FromDisk(db)

	events, err := c.Events(testHeight)
	assert.NoError(t, err)
	assert.Len(t, events, 2)

	_, err = c.Events(math.MaxUint64)
	assert.Error(t, err)
}

func TestDisk_Transactions(t *testing.T) {
	db := populatedDB(t)
	defer db.Close()
	c := chain.FromDisk(db)

	tt, err := c.Transactions(testHeight)
	assert.NoError(t, err)
	assert.Len(t, tt, 2)

	_, err = c.Transactions(math.MaxUint64)
	assert.Error(t, err)
}

func TestDisk_Collections(t *testing.T) {
	db := populatedDB(t)
	defer db.Close()
	c := chain.FromDisk(db)

	tt, err := c.Collections(testHeight)
	assert.NoError(t, err)
	assert.Len(t, tt, 2)

	_, err = c.Collections(math.MaxUint64)
	assert.Error(t, err)
}

func populatedDB(t *testing.T) *badger.DB {
	t.Helper()

	db := helpers.InMemoryDB(t)

	err := db.Update(func(tx *badger.Txn) error {
		err := operation.InsertRootHeight(testHeight)(tx)
		if err != nil {
			return err
		}

		err = operation.InsertHeader(testBlockID, &flow.Header{ChainID: testChainID})(tx)
		if err != nil {
			return err
		}

		err = operation.IndexBlockHeight(testHeight, testBlockID)(tx)
		if err != nil {
			return err
		}

		err = operation.IndexStateCommitment(testBlockID, testCommit)(tx)
		if err != nil {
			return err
		}

		events := []flow.Event{
			{
				Type:             "test",
				TransactionIndex: 1,
				EventIndex:       2,
			},
			{
				Type:             "test",
				TransactionIndex: 3,
				EventIndex:       4,
			},
		}
		err = operation.InsertEvent(testBlockID, events[0])(tx)
		if err != nil {
			return err
		}
		err = operation.InsertEvent(testBlockID, events[1])(tx)
		if err != nil {
			return err
		}

		tb1 := flow.TransactionBody{
			ReferenceBlockID: testBlockID,
			GasLimit:         42,
			Payer:            flow.Address{0x12, 0x12, 0x12, 0x12, 0x12, 0x12, 0x12, 0x12},
		}
		tb2 := flow.TransactionBody{
			ReferenceBlockID: testBlockID,
			GasLimit:         84,
			Payer:            flow.Address{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		}

		err = operation.InsertTransactionResult(testBlockID, &flow.TransactionResult{TransactionID: tb1.ID()})(tx)
		if err != nil {
			return err
		}

		err = operation.InsertTransactionResult(testBlockID, &flow.TransactionResult{TransactionID: tb2.ID()})(tx)
		if err != nil {
			return err
		}

		err = operation.InsertTransaction(tb1.ID(), &tb1)(tx)
		if err != nil {
			return err
		}
		err = operation.InsertTransaction(tb2.ID(), &tb2)(tx)
		if err != nil {
			return err
		}

		tb3 := flow.TransactionBody{
			ReferenceBlockID: testBlockID,
			GasLimit:         21,
			Payer:            flow.Address{0xb0, 0x20, 0xe8, 0x58, 0x72, 0xc8, 0x12, 0x59},
		}
		tb4 := flow.TransactionBody{
			ReferenceBlockID: testBlockID,
			GasLimit:         168,
			Payer:            flow.Address{0x94, 0x2f, 0x2f, 0xf3, 0x50, 0x6b, 0xa8, 0xde},
		}

		collection1 := flow.LightCollection{Transactions: []flow.Identifier{tb1.ID(), tb2.ID()}}
		collection2 := flow.LightCollection{Transactions: []flow.Identifier{tb3.ID(), tb4.ID()}}

		err = operation.IndexCollectionByTransaction(tb1.ID(), collection1.ID())(tx)
		if err != nil {
			return err
		}

		err = operation.IndexCollectionByTransaction(tb2.ID(), collection1.ID())(tx)
		if err != nil {
			return err
		}

		err = operation.IndexCollectionByTransaction(tb3.ID(), collection2.ID())(tx)
		if err != nil {
			return err
		}

		err = operation.IndexCollectionByTransaction(tb4.ID(), collection2.ID())(tx)
		if err != nil {
			return err
		}

		err = operation.InsertCollection(&collection1)(tx)
		if err != nil {
			return err
		}

		err = operation.InsertCollection(&collection2)(tx)
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(t, err)

	return db
}
