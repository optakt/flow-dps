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

package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/node"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"

	"github.com/optakt/flow-dps/testing/mocks"
)

func TestAllPaths(t *testing.T) {
	t.Run("nominal case with single path", func(t *testing.T) {
		t.Parallel()

		testNode := node.NewLeaf(mocks.GenericLedgerPaths[0], mocks.GenericLedgerPayloads[0], 256)
		testTrie, err := trie.NewMTrie(testNode)
		require.NoError(t, err)

		got := allPaths(testTrie)

		assert.NotEmpty(t, got)
		assert.Equal(t, []ledger.Path{mocks.GenericLedgerPaths[0]}, got)
	})

	t.Run("nominal case with multiple paths", func(t *testing.T) {
		t.Parallel()

		got := allPaths(mocks.GenericTrie)

		// Only the paths in nodes 1, 2 and 4 are taken into account since they are the only leaves.
		assert.Len(t, got, 3)
		assert.Equal(t, []ledger.Path{mocks.GenericLedgerPaths[3], mocks.GenericLedgerPaths[1], mocks.GenericLedgerPaths[0]}, got)
	})

	t.Run("empty trie", func(t *testing.T) {
		t.Parallel()

		got := allPaths(trie.NewEmptyMTrie())

		assert.Empty(t, got)
	})
}

func TestPathsPayloads(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		gotPaths, gotPayloads := pathsPayloads(mocks.GenericTrieUpdate)

		// Expect payloads from deduplicated paths.
		wantPayloads := []ledger.Payload{
			*mocks.GenericLedgerPayloads[4],
			*mocks.GenericLedgerPayloads[3],
			*mocks.GenericLedgerPayloads[0],
			*mocks.GenericLedgerPayloads[1],
			*mocks.GenericLedgerPayloads[5],
		}
		assert.Equal(t, wantPayloads, gotPayloads)
		for _, wantPath := range mocks.GenericLedgerPaths {
			assert.Contains(t, gotPaths, wantPath)
		}

		// Verify that paths are sorted and deduplicated.
		// mocks.GenericLedgerPaths[2] and mocks.GenericLedgerPaths[4]
		// are identical so the latter is omitted because of deduplication.
		sortedPaths := []ledger.Path{
			mocks.GenericLedgerPaths[2],
			mocks.GenericLedgerPaths[3],
			mocks.GenericLedgerPaths[0],
			mocks.GenericLedgerPaths[1],
			mocks.GenericLedgerPaths[5],
		}
		assert.Equalf(t, sortedPaths, gotPaths, "expected paths to be sorted alphabetically and deduplicated")
	})

	t.Run("nominal case with empty trie update", func(t *testing.T) {
		t.Parallel()

		emptyUpdate := &ledger.TrieUpdate{}

		gotPaths, gotPayloads := pathsPayloads(emptyUpdate)

		assert.Empty(t, gotPaths)
		assert.Empty(t, gotPayloads)
	})
}
