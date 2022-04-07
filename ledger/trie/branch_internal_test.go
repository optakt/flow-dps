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
	"github.com/onflow/flow-go/ledger/common/utils"
)

// Test_BranchWithoutChildren verifies that the hash value of a branch node without children is computed correctly.
// We test the hash at the lowest-possible height (0), at an interim height (9) and the max possible height (256)
func Test_BranchWithoutChildren(t *testing.T) {
	t.Skip() // Does not exist in our implementation
}

// Test_BranchWithOneChild verifies that the hash value of a branch node with
// only one child (left or right) is computed correctly.
func Test_BranchWithOneChild(t *testing.T) {
	t.Skip() // Does not exist in our implementation
}

// Test_BranchWithBothChildren verifies that the hash value of a branch node with
// both children (left and right) is computed correctly.
func Test_BranchWithBothChildren(t *testing.T) {
	const expectedHashHex = "1e4754fb35ec011b6192e205de403c1031d8ce64bd3d1ff8f534a20595af90c3"

	leftPath := utils.PathByUint16(56809)
	leftPayload := utils.LightPayload(56810, 59656)
	lHash, err := hash.ToHash(leftPath[:])
	require.NoError(t, err)
	leftChild := Leaf{
		clean: true,
		hash:  ledger.ComputeCompactValue(lHash, leftPayload.Value, 0),
	}

	rightPath := utils.PathByUint16(2)
	rightPayload := utils.LightPayload(11, 22)
	rHash, err := hash.ToHash(rightPath[:])
	require.NoError(t, err)
	rightChild := Leaf{
		clean: true,
		hash:  ledger.ComputeCompactValue(rHash, rightPayload.Value, 0),
	}

	n := Branch{left: &leftChild, right: &rightChild}
	got := n.Hash(semaphore.NewWeighted(1), 1)
	require.Equal(t, expectedHashHex, hex.EncodeToString(got[:]))
}
