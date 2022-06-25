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

package invoker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-dps/testing/mocks"

	"github.com/onflow/flow-go/ledger"
)

func TestReadRegister(t *testing.T) {
	owner := string(mocks.GenericLedgerKey.KeyParts[0].Value)
	controller := string(mocks.GenericLedgerKey.KeyParts[1].Value)
	key := string(mocks.GenericLedgerKey.KeyParts[2].Value)

	t.Run("nominal case with cached register", func(t *testing.T) {
		t.Parallel()

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			// Return that the cache contains the register's value already.
			return mocks.GenericBytes, true
		}

		var indexCalled bool
		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(uint64, []ledger.Path) ([]ledger.Value, error) {
			indexCalled = true
			return nil, nil
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		value, err := readFunc(owner, controller, key)

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
		index.ValuesFunc = func(uint64, []ledger.Path) ([]ledger.Value, error) {
			indexCalled = true
			return []ledger.Value{mocks.GenericBytes}, nil
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		value, err := readFunc(owner, controller, key)

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
		index.ValuesFunc = func(uint64, []ledger.Path) ([]ledger.Value, error) {
			return nil, mocks.GenericError
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		_, err := readFunc(owner, controller, key)

		assert.Error(t, err)
	})
}
