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
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/complete/mtrie/node"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
)

// Common test variables used for both pathings_test.go and transitions_test.go.
var (
	testPath1 = ledger.Path{
		0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70,
		0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1,
		0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70,
		0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1,
	}
	testPath2 = ledger.Path{
		0xd5, 0x08, 0x44, 0x13, 0xdb, 0xe5, 0x2b, 0xd2,
		0x3a, 0x66, 0x7f, 0xc4, 0x08, 0xe0, 0x54, 0x60,
		0xd5, 0x08, 0x44, 0x13, 0xdb, 0xe5, 0x2b, 0xd2,
		0x3a, 0x66, 0x7f, 0xc4, 0x08, 0xe0, 0x54, 0x60,
	}
	testPath3 = ledger.Path{
		0x60, 0x0a, 0xd8, 0xa4, 0xf1, 0x6b, 0xce, 0x2e,
		0x57, 0x59, 0xfd, 0x6e, 0x45, 0xcf, 0xa9, 0xa0,
		0x60, 0x0a, 0xd8, 0xa4, 0xf1, 0x6b, 0xce, 0x2e,
		0x57, 0x59, 0xfd, 0x6e, 0x45, 0xcf, 0xa9, 0xa0,
	}
	testPath4 = ledger.Path{
		0xa5, 0x68, 0x7b, 0x2d, 0x95, 0x18, 0x7b, 0xc7,
		0xce, 0xd0, 0xe1, 0x02, 0xd6, 0xce, 0xfe, 0x93,
		0xa5, 0x68, 0x7b, 0x2d, 0x95, 0x18, 0x7b, 0xc7,
		0xce, 0xd0, 0xe1, 0x02, 0xd6, 0xce, 0xfe, 0x93,
	}
	testPath5 = ledger.Path{
		0x60, 0x0a, 0xd8, 0xa4, 0xf1, 0x6b, 0xce, 0x2e,
		0x57, 0x59, 0xfd, 0x6e, 0x45, 0xcf, 0xa9, 0xa0,
		0x60, 0x0a, 0xd8, 0xa4, 0xf1, 0x6b, 0xce, 0x2e,
		0x57, 0x59, 0xfd, 0x6e, 0x45, 0xcf, 0xa9, 0xa0,
	}
	testPath6 = ledger.Path{
		0xfd, 0x84, 0xc0, 0xa7, 0xb2, 0x35, 0xc9, 0x89,
		0xc1, 0x8e, 0x6a, 0xa2, 0x69, 0x04, 0xfe, 0xba,
		0xfd, 0x84, 0xc0, 0xa7, 0xb2, 0x35, 0xc9, 0x89,
		0xc1, 0x8e, 0x6a, 0xa2, 0x69, 0x04, 0xfe, 0xba,
	}
	testKey = ledger.NewKey([]ledger.KeyPart{
		ledger.NewKeyPart(0, []byte(`owner`)),
		ledger.NewKeyPart(1, []byte(`controller`)),
		ledger.NewKeyPart(2, []byte(`key`)),
	})
	testValue1   = ledger.Value(`test1`)
	testValue2   = ledger.Value(`test2`)
	testValue3   = ledger.Value(`test3`)
	testValue4   = ledger.Value(`test4`)
	testValue5   = ledger.Value(`test5`)
	testValue6   = ledger.Value(`test6`)
	testPayload1 = ledger.NewPayload(testKey, testValue1)
	testPayload2 = ledger.NewPayload(testKey, testValue2)
	testPayload3 = ledger.NewPayload(testKey, testValue3)
	testPayload4 = ledger.NewPayload(testKey, testValue4)
	testPayload5 = ledger.NewPayload(testKey, testValue5)
	testPayload6 = ledger.NewPayload(testKey, testValue6)

	testPaths    = []ledger.Path{testPath1, testPath2, testPath2, testPath2, testPath3, testPath4}
	testPayloads = []*ledger.Payload{testPayload1, testPayload2, testPayload3, testPayload4, testPayload5, testPayload6}

	testUpdate = &ledger.TrieUpdate{
		Paths:    testPaths,
		Payloads: testPayloads,
	}

	// Test nodes visual representation:
	//           6 (root)
	//          / \
	//         3   5
	//        / \   \
	//       1   2   4
	//
	testNode1 = node.NewLeaf(testPath1, testPayload1, 256)
	testNode2 = node.NewLeaf(testPath2, testPayload2, 256)
	testNode3 = node.NewNode(256, testNode1, testNode2, testPath3, testPayload3, hash.DummyHash, 64, 64)
	testNode4 = node.NewLeaf(testPath4, testPayload4, 256)
	testNode5 = node.NewNode(256, testNode4, nil, testPath5, testPayload5, hash.DummyHash, 64, 64)
	testRoot  = node.NewNode(256, testNode3, testNode5, testPath6, testPayload6, hash.DummyHash, 64, 64)
)

func TestAllPaths(t *testing.T) {
	t.Run("nominal case with single path", func(t *testing.T) {
		t.Parallel()

		testNode := node.NewLeaf(testPath1, testPayload1, 256)
		testTrie, err := trie.NewMTrie(testNode)
		require.NoError(t, err)

		got := allPaths(testTrie)

		assert.NotEmpty(t, got)
		assert.Equal(t, []ledger.Path{testPath1}, got)
	})

	t.Run("nominal case with multiple paths", func(t *testing.T) {
		t.Parallel()

		testTrie, err := trie.NewMTrie(testRoot)
		require.NoError(t, err)

		got := allPaths(testTrie)

		// Only the paths in nodes 1, 2 and 4 are taken into account since they are the only leaves.
		assert.Len(t, got, 3)
		assert.Equal(t, []ledger.Path{testPath4, testPath2, testPath1}, got)
	})

	t.Run("empty trie", func(t *testing.T) {
		t.Parallel()

		testTrie := trie.NewEmptyMTrie()

		got := allPaths(testTrie)

		assert.Empty(t, got)
	})
}

func TestPathsPayloads(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		gotPaths, gotPayloads := pathsPayloads(testUpdate)

		// Expect payloads from deduplicated paths.
		wantPayloads := []ledger.Payload{*testPayload5, *testPayload6, *testPayload1, *testPayload4}
		assert.Equal(t, wantPayloads, gotPayloads)
		for _, wantPath := range testPaths {
			assert.Contains(t, gotPaths, wantPath)
		}

		// Verify that paths are sorted and deduplicated.
		sortedPaths := []ledger.Path{testPath3, testPath4, testPath1, testPath2}
		assert.Equalf(t, sortedPaths, gotPaths, "expected paths to be sorted alphabetically and deduplicated")
	})

	t.Run("nominal case with empty trie update", func(t *testing.T) {
		t.Parallel()

		emptyUpdate := &ledger.TrieUpdate{
			Paths:    []ledger.Path{},
			Payloads: []*ledger.Payload{},
		}

		gotPaths, gotPayloads := pathsPayloads(emptyUpdate)

		assert.Empty(t, gotPaths)
		assert.Empty(t, gotPayloads)
	})
}
