package initializer_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/state/protocol/inmem"
	"github.com/onflow/flow-go/storage/badger/operation"
	"github.com/onflow/flow-go/utils/unittest"

	"github.com/onflow/flow-archive/service/initializer"
	"github.com/onflow/flow-archive/testing/helpers"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestProtocolState(t *testing.T) {
	header := mocks.GenericHeader
	participants := unittest.CompleteIdentitySet()
	rootSnapshot := unittest.RootSnapshotFixture(participants).Encodable()
	data, err := json.Marshal(rootSnapshot)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		file := bytes.NewBuffer(data)

		err := initializer.ProtocolState(file, db)
		assert.NoError(t, err)
	})

	t.Run("handles already populated protocol state DB", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		root := header.Height + 1
		require.NoError(t, db.Update(operation.InsertRootHeight(root)))

		file := bytes.NewBuffer(data)

		err := initializer.ProtocolState(file, db)
		assert.NoError(t, err)

		var have uint64
		assert.NoError(t, db.View(operation.RetrieveRootHeight(&have)))
		assert.Equal(t, root, have)
	})

	t.Run("handles invalid snapshot encoding", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		err := initializer.ProtocolState(bytes.NewBuffer(mocks.GenericBytes), db)
		assert.Error(t, err)
	})

	t.Run("Panics on empty snapshot", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer func() {
			r := recover()
			assert.NotNil(t, r)
			db.Close()
		}()

		data, err := json.Marshal(&inmem.EncodableSnapshot{})
		require.NoError(t, err)

		reader := bytes.NewBuffer(data)

		_ = initializer.ProtocolState(reader, db)

	})
}
