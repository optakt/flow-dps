package initializer_test

import (
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/onflow/flow-archive/service/initializer"
	"github.com/onflow/flow-archive/testing/helpers"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestCatchupBlocks(t *testing.T) {
	rootHeight := uint64(0)
	toIndex := uint64(4)
	header := mocks.GenericHeader
	blockIDs := mocks.GenericBlockIDs(int(toIndex))

	t.Run("handles index not empty", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		lastHeight := header.Height - toIndex
		require.NoError(t, db.Update(operation.InsertFinalizedHeight(header.Height)))
		for i := uint64(1); i <= toIndex; i++ { // Start at one since we ignore the already indexed height.
			require.NoError(t, db.Update(operation.IndexBlockHeight(lastHeight+i, blockIDs[i-1])))
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return lastHeight, nil
		}

		got, err := initializer.CatchupBlocks(db, reader)

		require.NoError(t, err)
		assert.Equal(t, blockIDs, got)
	})

	t.Run("handles empty index", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.InsertRootHeight(rootHeight)))
		require.NoError(t, db.Update(operation.InsertFinalizedHeight(toIndex)))
		for i := rootHeight + 1; i <= rootHeight+toIndex; i++ {
			require.NoError(t, db.Update(operation.IndexBlockHeight(i, blockIDs[i-1])))
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return 0, badger.ErrKeyNotFound
		}

		got, err := initializer.CatchupBlocks(db, reader)

		require.NoError(t, err)
		assert.Equal(t, blockIDs, got)
	})

	t.Run("handles reader failure on Last", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.InsertRootHeight(rootHeight)))
		require.NoError(t, db.Update(operation.InsertFinalizedHeight(toIndex)))
		for i := rootHeight + 1; i <= rootHeight+toIndex; i++ {
			require.NoError(t, db.Update(operation.IndexBlockHeight(i, blockIDs[i-1])))
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		_, err := initializer.CatchupBlocks(db, reader)

		assert.Error(t, err)
	})

	t.Run("handles missing block height in database", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.InsertRootHeight(rootHeight)))
		require.NoError(t, db.Update(operation.InsertFinalizedHeight(toIndex)))

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return 0, badger.ErrKeyNotFound
		}

		_, err := initializer.CatchupBlocks(db, reader)

		assert.Error(t, err)
	})
}
