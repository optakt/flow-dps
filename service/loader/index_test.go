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

package loader_test

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"

	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/service/loader"
	"github.com/optakt/flow-dps/service/storage"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestLoader_FromIndex(t *testing.T) {
	entries := 5
	paths := mocks.GenericLedgerPaths(entries)
	payloads := mocks.GenericLedgerPayloads(entries)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		log := zerolog.Nop()
		codec := zbor.NewCodec()
		storage := storage.New(codec)

		db := helpers.InMemoryDB(t)
		defer db.Close()

		for i := 0; i < entries; i++ {
			height := mocks.GenericHeight + uint64(i)
			require.NoError(t, db.Update(storage.SavePayload(height, paths[i], payloads[i])))
		}

		load := loader.FromIndex(log, storage, db)

		trie, err := load.Trie()
		require.NoError(t, err)
		assert.True(t, trie.IsAValidTrie())
	})

	t.Run("handles failure to initialize trie", func(t *testing.T) {
		t.Parallel()

		log := zerolog.Nop()
		codec := zbor.NewCodec()
		storage := storage.New(codec)

		db := helpers.InMemoryDB(t)
		defer db.Close()

		initializer := mocks.BaselineLoader(t)
		initializer.TrieFunc = func() (*trie.MTrie, error) {
			return nil, mocks.GenericError
		}

		load := loader.FromIndex(log, storage, db, loader.WithInitializer(initializer))

		_, err := load.Trie()
		require.Error(t, err)
	})

	t.Run("handles failure to iterate ledger", func(t *testing.T) {
		t.Parallel()

		log := zerolog.Nop()
		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func([]byte, interface{}) error {
			return mocks.GenericError
		}
		storage := storage.New(codec)

		db := helpers.InMemoryDB(t)
		defer db.Close()

		for i := 0; i < entries; i++ {
			height := mocks.GenericHeight + uint64(i)
			require.NoError(t, db.Update(storage.SavePayload(height, paths[i], payloads[i])))
		}

		load := loader.FromIndex(log, storage, db)

		_, err := load.Trie()
		require.Error(t, err)
	})
}
