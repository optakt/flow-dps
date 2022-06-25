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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/service/chain"
	"github.com/onflow/flow-dps/testing/helpers"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestDisk_Root(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.InsertRootHeight(mocks.GenericHeight)))

		c := chain.FromDisk(db)

		root, err := c.Root()

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, root)
	})

	t.Run("handles missing root height entry in db", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Root()

		assert.Error(t, err)
	})
}

func TestDisk_Header(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))
		require.NoError(t, db.Update(operation.InsertHeader(mocks.GenericHeader.ID(), mocks.GenericHeader)))

		c := chain.FromDisk(db)

		header, err := c.Header(mocks.GenericHeight)

		require.NoError(t, err)
		require.NotNil(t, header)
		assert.Equal(t, dps.FlowTestnet, header.ChainID)

		_, err = c.Header(math.MaxUint64)

		assert.Error(t, err)
	})

	t.Run("handles missing entry for indexed height", func(t *testing.T) {
		t.Parallel()

		// Only index the block height, but no header for that height.
		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))

		c := chain.FromDisk(db)

		_, err := c.Header(mocks.GenericHeight)

		assert.Error(t, err)
	})

	t.Run("handles call on non-indexed height", func(t *testing.T) {
		t.Parallel()

		// Use an empty DB without any entries.
		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Header(mocks.GenericHeight)

		assert.Error(t, err)
	})

	t.Run("returns ErrFinished when no more entries are in the DB", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Header(mocks.GenericHeight)

		assert.Equal(t, dps.ErrFinished, err)
	})
}

func TestDisk_Commit(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))
		require.NoError(t, db.Update(operation.IndexStateCommitment(mocks.GenericHeader.ID(), mocks.GenericCommit(0))))

		c := chain.FromDisk(db)

		commit, err := c.Commit(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericCommit(0), commit)

		_, err = c.Commit(math.MaxUint64)

		assert.Error(t, err)
	})

	t.Run("returns db error when block exists but commit is missing", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))

		c := chain.FromDisk(db)

		_, err := c.Commit(mocks.GenericHeight)

		assert.Error(t, err)
		assert.NotEqual(t, dps.ErrFinished, err)
	})
}

func TestDisk_Events(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))
		require.NoError(t, db.Update(operation.InsertEvent(mocks.GenericHeader.ID(), mocks.GenericEvent(0))))
		require.NoError(t, db.Update(operation.InsertEvent(mocks.GenericHeader.ID(), mocks.GenericEvent(1))))

		c := chain.FromDisk(db)

		events, err := c.Events(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Len(t, events, 2)
		assert.Contains(t, events, mocks.GenericEvent(0))
		assert.Contains(t, events, mocks.GenericEvent(1))

		_, err = c.Events(math.MaxUint64)

		assert.Error(t, err)
	})

	t.Run("handles call on non-indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Events(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestDisk_Collections(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))
		require.NoError(t, db.Update(operation.IndexPayloadGuarantees(mocks.GenericHeader.ID(), mocks.GenericCollectionIDs(2))))
		require.NoError(t, db.Update(operation.InsertCollection(mocks.GenericCollection(0))))
		require.NoError(t, db.Update(operation.InsertCollection(mocks.GenericCollection(1))))

		c := chain.FromDisk(db)

		tt, err := c.Collections(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Len(t, tt, 2)

		_, err = c.Collections(math.MaxUint64)

		assert.Error(t, err)
	})

	t.Run("handles missing entry for indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))

		c := chain.FromDisk(db)

		_, err := c.Collections(mocks.GenericHeight)

		assert.Error(t, err)
	})

	t.Run("handles call on non-indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Collections(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestDisk_Guarantees(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))
		require.NoError(t, db.Update(operation.IndexPayloadGuarantees(mocks.GenericHeader.ID(), mocks.GenericCollectionIDs(2))))
		require.NoError(t, db.Update(operation.InsertGuarantee(mocks.GenericCollection(0).ID(), mocks.GenericGuarantee(0))))
		require.NoError(t, db.Update(operation.InsertGuarantee(mocks.GenericCollection(1).ID(), mocks.GenericGuarantee(1))))

		c := chain.FromDisk(db)

		tt, err := c.Guarantees(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Len(t, tt, 2)

		_, err = c.Guarantees(math.MaxUint64)

		assert.Error(t, err)
	})

	t.Run("handles missing entry for indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))

		c := chain.FromDisk(db)

		_, err := c.Guarantees(mocks.GenericHeight)

		assert.Error(t, err)
	})

	t.Run("handles call on non-indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Guarantees(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestDisk_Transactions(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))
		require.NoError(t, db.Update(operation.IndexPayloadGuarantees(mocks.GenericHeader.ID(), mocks.GenericCollectionIDs(2))))
		require.NoError(t, db.Update(operation.InsertCollection(mocks.GenericCollection(0))))
		require.NoError(t, db.Update(operation.InsertTransaction(mocks.GenericTransaction(0).ID(), mocks.GenericTransaction(0))))
		require.NoError(t, db.Update(operation.InsertTransaction(mocks.GenericTransaction(1).ID(), mocks.GenericTransaction(1))))
		require.NoError(t, db.Update(operation.InsertCollection(mocks.GenericCollection(1))))
		require.NoError(t, db.Update(operation.InsertTransaction(mocks.GenericTransaction(2).ID(), mocks.GenericTransaction(2))))
		require.NoError(t, db.Update(operation.InsertTransaction(mocks.GenericTransaction(3).ID(), mocks.GenericTransaction(3))))

		c := chain.FromDisk(db)

		tt, err := c.Transactions(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Len(t, tt, 4)

		_, err = c.Transactions(math.MaxUint64)

		assert.Error(t, err)
	})

	t.Run("handles missing entry for indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))

		c := chain.FromDisk(db)

		_, err := c.Transactions(mocks.GenericHeight)

		assert.Error(t, err)
	})

	t.Run("handles call on non-indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Transactions(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestDisk_TransactionResults(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))
		require.NoError(t, db.Update(operation.InsertTransactionResult(mocks.GenericHeader.ID(), mocks.GenericResult(0))))
		require.NoError(t, db.Update(operation.InsertTransactionResult(mocks.GenericHeader.ID(), mocks.GenericResult(1))))
		require.NoError(t, db.Update(operation.InsertTransactionResult(mocks.GenericHeader.ID(), mocks.GenericResult(2))))
		require.NoError(t, db.Update(operation.InsertTransactionResult(mocks.GenericHeader.ID(), mocks.GenericResult(3))))

		c := chain.FromDisk(db)

		tr, err := c.Results(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Len(t, tr, 4)

		_, err = c.Results(math.MaxUint64)

		assert.Error(t, err)
	})

	t.Run("handles call on non-indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Results(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestDisk_Seals(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))
		require.NoError(t, db.Update(operation.IndexPayloadSeals(mocks.GenericHeader.ID(), mocks.GenericSealIDs(4))))
		require.NoError(t, db.Update(operation.InsertSeal(mocks.GenericSeal(0).ID(), mocks.GenericSeal(0))))
		require.NoError(t, db.Update(operation.InsertSeal(mocks.GenericSeal(1).ID(), mocks.GenericSeal(1))))
		require.NoError(t, db.Update(operation.InsertSeal(mocks.GenericSeal(2).ID(), mocks.GenericSeal(2))))
		require.NoError(t, db.Update(operation.InsertSeal(mocks.GenericSeal(3).ID(), mocks.GenericSeal(3))))

		c := chain.FromDisk(db)

		seals, err := c.Seals(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Len(t, seals, 4)

		_, err = c.Seals(math.MaxUint64)

		assert.Error(t, err)
	})

	t.Run("handles missing entry for indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.IndexBlockHeight(mocks.GenericHeight, mocks.GenericHeader.ID())))

		c := chain.FromDisk(db)

		_, err := c.Seals(mocks.GenericHeight)

		assert.Error(t, err)
	})

	t.Run("handles call on non-indexed height", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		c := chain.FromDisk(db)

		_, err := c.Seals(mocks.GenericHeight)

		assert.Error(t, err)
	})
}
