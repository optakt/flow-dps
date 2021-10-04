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
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger/complete/mtrie"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/module/metrics"

	"github.com/optakt/flow-dps/service/loader"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestLoader_FromCheckpoint(t *testing.T) {

	t.Run("nominal case", func(t *testing.T) {

		forest, err := mtrie.NewForest(1, &metrics.NoopCollector{}, func(tree *trie.MTrie) error { return nil })
		require.NoError(t, err)

		trie := mocks.GenericTrie

		err = forest.AddTrie(trie)
		require.NoError(t, err)

		flattened, err := flattener.FlattenForest(forest)
		require.NoError(t, err)

		buffer := bytes.Buffer{}
		err = wal.StoreCheckpoint(flattened, &buffer)
		require.NoError(t, err)

		checkpoint := bytes.NewReader(buffer.Bytes())

		load := loader.FromCheckpoint(checkpoint)

		loadedTrie, err := load.Trie()
		require.NoError(t, err)

		eq := loadedTrie.Equals(trie)
		require.True(t, eq)
	})

	t.Run("handles failure to read checkpoint", func(t *testing.T) {

		// Create a reader for a malformed checkpoint.
		reader := bytes.NewReader(mocks.GenericBytes)
		load := loader.FromCheckpoint(reader)

		_, err := load.Trie()
		require.Error(t, err)
	})

	t.Run("handles failure to rebuild tries", func(t *testing.T) {

		forest := &flattener.FlattenedForest{
			Nodes: []*flattener.StorableNode{
				{},
			},
			Tries: []*flattener.StorableTrie{
				{},
			},
		}

		buffer := bytes.Buffer{}
		err := wal.StoreCheckpoint(forest, &buffer)
		require.NoError(t, err)

		checkpoint := bytes.NewReader(buffer.Bytes())

		load := loader.FromCheckpoint(checkpoint)

		_, err = load.Trie()
		require.Error(t, err)
	})

	t.Run("handles failure with multiple tries in root checkpoint", func(t *testing.T) {

		// Create a forest with capacity for two tries.
		// This will trigger an error since the root checkpoint should have only one.
		forest, err := mtrie.NewForest(2, &metrics.NoopCollector{}, func(tree *trie.MTrie) error { return nil })
		require.NoError(t, err)

		trie := mocks.GenericTrie

		err = forest.AddTrie(trie)
		require.NoError(t, err)

		flattened, err := flattener.FlattenForest(forest)
		require.NoError(t, err)

		buffer := bytes.Buffer{}
		err = wal.StoreCheckpoint(flattened, &buffer)
		require.NoError(t, err)

		checkpoint := bytes.NewReader(buffer.Bytes())
		load := loader.FromCheckpoint(checkpoint)

		_, err = load.Trie()
		require.Error(t, err)
	})

}
