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

	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestExtension(t *testing.T) {
	testNode := trie.NewLeaf(0, mocks.GenericLedgerPath(0), mocks.GenericLedgerPayload(0))

	tests := []struct {
		name   string
		height uint16
		skip   uint16

		wantHash string
	}{
		{
			name:   "skip 242 height",
			height: 256,
			skip:   14,

			wantHash: "b873fbe1c141397e361d434d57690c44ca46ab0fccd010044d681b8149c2e46a",
		},
		{
			name:   "skip 1 height",
			height: 15,
			skip:   14,

			wantHash: "aa62454a6a763f993f1424efe8235e9e36d718ceafc5d8be73d9e52c6df85b98",
		},
		{
			name:   "skip no height",
			height: 15,
			skip:   15,

			wantHash: "33e7a8127af4dbfc7c4ce433607a3fec2d63fddcfdec1b82060985e663b53e4b",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			n := trie.NewExtension(
				test.height,
				test.skip,
				mocks.GenericLedgerPath(0),
				testNode,
				testNode,
			)

			got := n.Hash()
			require.Equal(t, test.wantHash, hex.EncodeToString(got[:]))
		})
	}
}
