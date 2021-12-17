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

package trie_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger/common/utils"
	"github.com/optakt/flow-dps/ledger/trie"
)

// Test_BranchWithoutChildren verifies that the hash value of a branch node without children is computed correctly.
// We test the hash at the lowest-possible height (0), at an interim height (9) and the max possible height (256)
func Test_BranchWithoutChildren(t *testing.T) {

	var expectedHashes = []string{
		"18373b4b038cbbf37456c33941a7e346e752acd8fafa896933d4859002b62619",
		"a37f98dbac56e315fbd4b9f9bc85fbd1b138ed4ae453b128c22c99401495af6d",
		"6e24e2397f130d9d17bef32b19a77b8f5bcf03fb7e9e75fd89b8a455675d574a",
	}

	n := trie.NewBranch(0, nil, nil)
	got := n.Hash()
	require.Equal(t, expectedHashes[0], hex.EncodeToString(got[:]))

	n = trie.NewBranch(9, nil, nil)
	got = n.Hash()
	require.Equal(t, expectedHashes[1], hex.EncodeToString(got[:]))

	n = trie.NewBranch(16, nil, nil)
	got = n.Hash()
	require.Equal(t, expectedHashes[2], hex.EncodeToString(got[:]))
}

// Test_BranchWithOneChild verifies that the hash value of a branch node with
// only one child (left or right) is computed correctly.
func Test_BranchWithOneChild(t *testing.T) {

	var expectedHashes = []string{
		"aa496f68adbbf43197f7e4b6ba1a63a47b9ce19b1587ca9ce587a7f29cad57d5",
		"9845f2c9e9c067ec6efba06ffb7c1be387b2a893ae979b1f6cb091bda1b7e12d",
	}

	path := utils.PathByUint16(56809)
	payload := utils.LightPayload(56810, 59656)
	c := trie.NewLeaf(0, path, payload)

	n := trie.NewBranch(1, c, nil)
	got := n.Hash()
	require.Equal(t, expectedHashes[0], hex.EncodeToString(got[:]))

	n = trie.NewBranch(1, nil, c)
	got = n.Hash()
	require.Equal(t, expectedHashes[1], hex.EncodeToString(got[:]))
}

// Test_BranchWithBothChildren verifies that the hash value of a branch node with
// both children (left and right) is computed correctly.
func Test_BranchWithBothChildren(t *testing.T) {

	const expectedHashHex = "1e4754fb35ec011b6192e205de403c1031d8ce64bd3d1ff8f534a20595af90c3"

	leftPath := utils.PathByUint16(56809)
	leftPayload := utils.LightPayload(56810, 59656)
	leftChild := trie.NewLeaf(0, leftPath, leftPayload)

	rightPath := utils.PathByUint16(2)
	rightPayload := utils.LightPayload(11, 22)
	rightChild := trie.NewLeaf(0, rightPath, rightPayload)

	n := trie.NewBranch(1, leftChild, rightChild)
	got := n.Hash()
	require.Equal(t, expectedHashHex, hex.EncodeToString(got[:]))
}
