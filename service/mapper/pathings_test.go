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
	"bytes"
	"sort"
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

		testNode := node.NewLeaf(mocks.GenericLedgerPath(0), mocks.GenericLedgerPayload(0), 256)
		testTrie, err := trie.NewMTrie(testNode)
		require.NoError(t, err)

		got := allPaths(testTrie)

		assert.NotEmpty(t, got)
		assert.Equal(t, []ledger.Path{mocks.GenericLedgerPath(0)}, got)
	})

	t.Run("nominal case with multiple paths", func(t *testing.T) {
		t.Parallel()

		got := allPaths(mocks.GenericTrie())

		// Only the paths in nodes 0, 1 and 3 are taken into account since they are the only leaves.
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
		testUpdate := mocks.GenericTrieUpdate()
		testPaths := []ledger.Path{
			testUpdate.Paths[0],
			testUpdate.Paths[0],
			testUpdate.Paths[1],
			testUpdate.Paths[2],
			testUpdate.Paths[3],
			testUpdate.Paths[4],
		}
		testPayloads := []*ledger.Payload{
			testUpdate.Payloads[0],
			testUpdate.Payloads[0],
			testUpdate.Payloads[1],
			testUpdate.Payloads[2],
			testUpdate.Payloads[3],
			testUpdate.Payloads[4],
		}
		testUpdate.Paths = testPaths
		testUpdate.Payloads = testPayloads

		gotPaths, gotPayloads := pathsPayloads(testUpdate)

		// Expect paths to be sorted.
		wantPaths := []ledger.Path{
			testUpdate.Paths[1],
			testUpdate.Paths[2],
			testUpdate.Paths[3],
			testUpdate.Paths[4],
			testUpdate.Paths[5],
		}
		sort.Slice(wantPaths, func(i, j int) bool {
			return bytes.Compare(wantPaths[i][:], wantPaths[j][:]) < 0
		})

		assert.Len(t, gotPayloads, 5)

		// Verify that paths are sorted and deduplicated.
		assert.Equalf(t, wantPaths, gotPaths, "expected paths to be sorted alphabetically and deduplicated")
	})

	t.Run("nominal case with empty trie update", func(t *testing.T) {
		t.Parallel()

		emptyUpdate := &ledger.TrieUpdate{}

		gotPaths, gotPayloads := pathsPayloads(emptyUpdate)

		assert.Empty(t, gotPaths)
		assert.Empty(t, gotPayloads)
	})
}
