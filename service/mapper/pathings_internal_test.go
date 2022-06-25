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

	"github.com/onflow/flow-dps/testing/mocks"
)

func TestAllPaths(t *testing.T) {
	t.Run("nominal case with single path", func(t *testing.T) {
		t.Parallel()

		testNode := node.NewLeaf(mocks.GenericLedgerPath(0), mocks.GenericLedgerPayload(0), 256)
		testTrie, err := trie.NewMTrie(testNode)
		require.NoError(t, err)

		got := allPaths(testTrie)

		assert.NotEmpty(t, got)
		assert.Equal(t, []ledger.Path{mocks.GenericLedgerPath(0)}, got)
	})

	t.Run("nominal case with multiple paths", func(t *testing.T) {
		t.Parallel()

		got := allPaths(mocks.GenericTrie)

		// Only the paths in nodes 1, 2 and 4 are taken into account since they are the only leaves.
		assert.Len(t, got, 3)
		assert.Equal(t, []ledger.Path{mocks.GenericLedgerPath(3), mocks.GenericLedgerPath(1), mocks.GenericLedgerPath(0)}, got)
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

		// Forge test update with duplicate and unsorted paths.
		testUpdate := mocks.GenericTrieUpdate(0)
		testPaths := mocks.GenericLedgerPaths(6)
		testUpdate.Paths = []ledger.Path{
			testPaths[0],
			testPaths[0],
			testPaths[1],
			testPaths[2],
			testPaths[3],
			testPaths[4],
		}

		gotPaths, gotPayloads := pathsPayloads(testUpdate)

		// Expect payloads from deduplicated paths.
		wantPayloads := []ledger.Payload{
			*mocks.GenericLedgerPayload(3),
			*mocks.GenericLedgerPayload(4),
			*mocks.GenericLedgerPayload(5),
			*mocks.GenericLedgerPayload(1),
			*mocks.GenericLedgerPayload(2),
		}
		assert.Equal(t, wantPayloads, gotPayloads)

		// Verify that paths are sorted and deduplicated.
		sortedPaths := []ledger.Path{
			testPaths[2],
			testPaths[3],
			testPaths[4],
			testPaths[0],
			testPaths[1],
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
