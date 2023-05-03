package invoker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-archive/testing/mocks"

	"github.com/onflow/flow-go/model/flow"
)

func TestReadRegister(t *testing.T) {
	owner := string(mocks.GenericLedgerKey.KeyParts[0].Value)
	key := string(mocks.GenericLedgerKey.KeyParts[1].Value)
	registerId := flow.RegisterID{
		Owner: owner,
		Key:   key,
	}

	t.Run("nominal case with cached register", func(t *testing.T) {
		t.Parallel()

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			// Return that the cache contains the register's value already.
			return mocks.GenericBytes, true
		}

		var indexCalled bool
		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(uint64, flow.RegisterIDs) ([]flow.RegisterValue, error) {
			indexCalled = true
			return nil, nil
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		value, err := readFunc(registerId)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericBytes, value[:])
		assert.False(t, indexCalled)
	})

	t.Run("nominal case without cached register", func(t *testing.T) {
		t.Parallel()

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			// Return that the cache DOES NOT contain the register's value already.
			return nil, false
		}

		var indexCalled bool
		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(uint64, flow.RegisterIDs) ([]flow.RegisterValue, error) {
			indexCalled = true
			return []flow.RegisterValue{mocks.GenericBytes}, nil
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		value, err := readFunc(registerId)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericBytes, value[:])
		assert.True(t, indexCalled)
	})

	t.Run("handles indexer failure on Values", func(t *testing.T) {
		t.Parallel()

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			// Return that the cache DOES NOT contain the register's value already.
			return nil, false
		}

		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(uint64, flow.RegisterIDs) ([]flow.RegisterValue, error) {
			return nil, mocks.GenericError
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		_, err := readFunc(registerId)

		assert.Error(t, err)
	})
}
