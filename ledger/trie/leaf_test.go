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

func Test_ProperLeaf(t *testing.T) {
	path := utils.PathByUint16(56809)
	payload := utils.LightPayload(56810, 59656)
	n := trie.NewLeaf(0, path, payload)
	expectedRootHashHex := "0ee164bc69981088186b5ceeb666e90e8e11bb15a1427aa56f47a484aedf73b4"
	got := n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_CompactLeaf(t *testing.T) {
	path := utils.PathByUint16(56809)
	payload := utils.LightPayload(56810, 59656)

	n := trie.NewLeaf(1, path, payload)
	expectedRootHashHex := "aa496f68adbbf43197f7e4b6ba1a63a47b9ce19b1587ca9ce587a7f29cad57d5"
	got := n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))

	n = trie.NewLeaf(9, path, payload)
	expectedRootHashHex = "606aa23fdc40443de85b75768b847f94ff1d726e0bafde037833fe27543bb988"
	got = n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))

	n = trie.NewLeaf(256, path, payload)
	expectedRootHashHex = "d2536303495a9325037d247cbb2b9be4d6cb3465986ea2c4481d8770ff16b6b0"
	got = n.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}
