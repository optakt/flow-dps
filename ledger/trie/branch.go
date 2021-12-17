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
	"github.com/onflow/flow-go/ledger/common/hash"
)

// Branch is a node is an intermediary node which has children.
// It does not need to contain a path, because its children are ordered
// based on their own path differences.
type Branch struct {
	lChild Node
	rChild Node

	height uint16
	hash   hash.Hash

	// dirty marks whether the current hash value of the branch is valid.
	// If this is set to true, the hash needs to be recomputed.
	dirty bool
}

// NewBranch creates a new branch with the given children at the given height.
func NewBranch(height uint16, lChild, rChild Node) *Branch {
	b := Branch{
		lChild: lChild,
		rChild: rChild,

		height: height,
		dirty:  true,
	}

	return &b
}

// computeHash computes the branch hash by hashing its children.
func (b *Branch) computeHash() {
	if b.lChild == nil && b.rChild == nil {
		b.hash = ledger.GetDefaultHashForHeight(int(b.height))
		return
	}

	var lHash, rHash hash.Hash
	if b.lChild != nil {
		lHash = b.lChild.Hash()
	} else {
		lHash = ledger.GetDefaultHashForHeight(int(b.height) - 1)
	}

	if b.rChild != nil {
		rHash = b.rChild.Hash()
	} else {
		rHash = ledger.GetDefaultHashForHeight(int(b.height) - 1)
	}

	b.hash = hash.HashInterNode(lHash, rHash)
	b.dirty = false
}

// Hash returns the branch hash. If it is currently dirty, it is recomputed first.
func (b *Branch) Hash() hash.Hash {
	if b.dirty {
		b.computeHash()
	}
	return b.hash
}

// FlagDirty flags the branch as having a dirty hash.
func (b *Branch) FlagDirty() {
	b.dirty = true
}

// Height returns the branch height.
func (b *Branch) Height() uint16 {
	return b.height
}

// Path returns a dummy path, since branches do not have paths.
func (b *Branch) Path() ledger.Path {
	return ledger.DummyPath
}

// LeftChild returns the left child.
func (b *Branch) LeftChild() Node {
	return b.lChild
}

// RightChild returns the right child.
func (b *Branch) RightChild() Node {
	return b.rChild
}
