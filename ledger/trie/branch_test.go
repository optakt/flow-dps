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

func Test_BranchWithoutChildren(t *testing.T) {
	n := trie.NewBranch(0, nil, nil)
	expectedRootHashHex := "18373b4b038cbbf37456c33941a7e346e752acd8fafa896933d4859002b62619"
	got := n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))

	n = trie.NewBranch(9, nil, nil)
	expectedRootHashHex = "a37f98dbac56e315fbd4b9f9bc85fbd1b138ed4ae453b128c22c99401495af6d"
	got = n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))

	n = trie.NewBranch(16, nil, nil)
	expectedRootHashHex = "6e24e2397f130d9d17bef32b19a77b8f5bcf03fb7e9e75fd89b8a455675d574a"
	got = n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_BranchWithOneChild(t *testing.T) {
	path := utils.PathByUint16(56809)
	payload := utils.LightPayload(56810, 59656)
	c := trie.NewLeaf(0, path, payload)

	n := trie.NewBranch(1, c, nil)
	expectedRootHashHex := "aa496f68adbbf43197f7e4b6ba1a63a47b9ce19b1587ca9ce587a7f29cad57d5"
	got := n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))

	n = trie.NewBranch(1, nil, c)
	expectedRootHashHex = "9845f2c9e9c067ec6efba06ffb7c1be387b2a893ae979b1f6cb091bda1b7e12d"
	got = n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_BranchWithBothChildren(t *testing.T) {
	leftPath := utils.PathByUint16(56809)
	leftPayload := utils.LightPayload(56810, 59656)
	leftChild := trie.NewLeaf(0, leftPath, leftPayload)

	rightPath := utils.PathByUint16(2)
	rightPayload := utils.LightPayload(11, 22)
	rightChild := trie.NewLeaf(0, rightPath, rightPayload)

	n := trie.NewBranch(1, leftChild, rightChild)
	expectedRootHashHex := "1e4754fb35ec011b6192e205de403c1031d8ce64bd3d1ff8f534a20595af90c3"
	got := n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}
