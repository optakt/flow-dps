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

package trie

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/optakt/flow-dps/testing/mocks"
)

// TestExtension verifies that the hash value of a branch node with
// both children (left and right) is computed correctly. As it is in our implementation,
// extensions can never have less than two children, so no further test is necessary.
func TestExtension(t *testing.T) {
	t.Skip()

	payload := mocks.GenericLedgerPayload(0)
	path := mocks.GenericLedgerPath(0)
	hash, err := hash.ToHash(path[:])
	require.NoError(t, err)

	testLeaf := Leaf{
		hash: ledger.ComputeCompactValue(hash, payload.Value, 0),
	}
	testBranch := Branch{
		left:  &testLeaf,
		right: &testLeaf,
	}

	tests := []struct {
		name   string
		height uint8
		count  uint8
		child  Node

		wantHash string
	}{
		{
			name:   "hash is leaf hash when child is leaf",
			height: 255,
			count:  0,
			child:  &testLeaf,

			wantHash: "d9a9d6f81bc7a638e30dda97404f13b6800702d58d099ffc12d56fca1e04dac4",
		},
		{
			name:   "branch child 255->14",
			height: 255,
			count:  241,
			child:  &testBranch,

			wantHash: "b873fbe1c141397e361d434d57690c44ca46ab0fccd010044d681b8149c2e46a",
		},
		{
			name:   "branch child 255->13",
			height: 255,
			count:  242,
			child:  &testBranch,

			wantHash: "17cda263a81b8395e81f49eb62689c44ae4b83a74521bfc201c60324f9e7b862",
		},
		{
			name:   "branch child 14->13",
			height: 14,
			count:  1,
			child:  &testBranch,

			wantHash: "1da5db03b0074f1ec5766755d4c402f98e1a379034ed9596bafa9f09dd2ce530",
		},
		{
			name:   "branch child 14->14",
			height: 14,
			count:  0,
			child:  &testBranch,

			wantHash: "aa62454a6a763f993f1424efe8235e9e36d718ceafc5d8be73d9e52c6df85b98",
		},
		{
			name:   "branch child 255->255",
			height: 255,
			count:  0,
			child:  &testBranch,

			wantHash: "bc093e283d158b37310df5871eed4359c694caba91afb7fdf35d76e588b511ec",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			n := Extension{
				clean: false,
				path:  &path,
				count: test.count,
				child: test.child,
			}

			got := n.Hash(semaphore.NewWeighted(1), int(test.height))
			require.Equal(t, test.wantHash, hex.EncodeToString(got[:]))
		})
	}
}
