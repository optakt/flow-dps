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
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/hash"
)

// Branch is a node is an intermediary node which has children.
// It does not need to contain a path, because its children are ordered
// based on their own path differences.
type Branch struct {
	hash  hash.Hash
	dirty bool
	left  Node
	right Node
}

// Hash returns the branch hash. If it is currently dirty, it is recomputed first.
func (b *Branch) Hash(height uint8, path [32]byte, getPayload payloadRetriever) [32]byte {
	if b.dirty {
		b.computeHash(height, path, getPayload)
	}
	return b.hash
}

// computeHash computes the branch hash by hashing its children.
func (b *Branch) computeHash(height uint8, path [32]byte, getPayload payloadRetriever) {
	if b.left == nil && b.right == nil {
		panic("branch node should never have empty children")
	}

	var lPath [32]byte
	copy(lPath[:], path[:])
	depth := ledger.NodeMaxHeight - 1 - height
	bitutils.SetBit(path[:], int(depth))
	lHash := b.left.Hash(height-1, lPath, getPayload)

	var rPath [32]byte
	copy(rPath[:], path[:])
	rHash := b.right.Hash(height-1, rPath, getPayload)

	b.hash = hash.HashInterNode(lHash, rHash)
	b.dirty = false
}
