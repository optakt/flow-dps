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

// Extension acts as a shortcut between many layers of the trie. It replaces a set of branches.
// The Flow implementation does not use extensions. This is a DPS optimization, which allows saving
// memory usage by reducing the amount of nodes necessary in the trie.
type Extension struct {
	lChild Node
	rChild Node

	height uint16
	// skip is the height up to which the extension skips to.
	skip uint16
	// path is the path that the extension skips through.
	// TODO: Store only skipped part of the path rather than whole path.
	// 	https://github.com/optakt/flow-dps/issues/516
	path ledger.Path
	hash hash.Hash

	// dirty marks whether the current hash value of the branch is valid.
	// If this is set to false, the hash needs to be recomputed.
	dirty bool
}

// NewExtension creates a new extension, from the given height to the given skip value.
func NewExtension(height, skip uint16, path ledger.Path, lChild, rChild Node) *Extension {
	e := Extension{
		lChild: lChild,
		rChild: rChild,

		height: height,
		skip:   skip,
		path:   path,
		dirty:  true,
	}

	return &e
}

// computeHash computes the extension's hash.
func (e *Extension) computeHash() {
	var computed hash.Hash

	// Compute the bottom hash.
	var lHash, rHash hash.Hash
	if e.lChild != nil {
		lHash = e.lChild.Hash()
	} else {
		lHash = ledger.GetDefaultHashForHeight(int(e.skip) - 1)
	}

	if e.rChild != nil {
		rHash = e.rChild.Hash()
	} else {
		rHash = ledger.GetDefaultHashForHeight(int(e.skip) - 1)
	}
	computed = hash.HashInterNode(lHash, rHash)

	// For each skipped height, combine the previous hash with the default ledger
	// height of the current layer.
	for i := e.skip; i < e.height; i++ {
		if bitutils.Bit(e.path[:], int(nodeHeight(i+1))) == 0 {
			lHash = computed
			rHash = ledger.GetDefaultHashForHeight(int(i))
		} else {
			lHash = ledger.GetDefaultHashForHeight(int(i))
			rHash = computed
		}
		computed = hash.HashInterNode(lHash, rHash)
	}

	e.hash = computed
	e.dirty = false
}

// Hash returns the extension hash. If it is currently dirty, it is recomputed first.
func (e *Extension) Hash() hash.Hash {
	if e.dirty {
		e.computeHash()
	}
	return e.hash
}

// FlagDirty flags the extension as having a dirty hash.
func (e *Extension) FlagDirty() {
	e.dirty = true
}

// Height returns the extension height.
func (e *Extension) Height() uint16 {
	return e.height
}

// Path returns the extension path.
func (e *Extension) Path() ledger.Path {
	return e.path
}

// LeftChild returns the left child.
func (e *Extension) LeftChild() Node {
	return e.lChild
}

// RightChild returns the right child.
func (e *Extension) RightChild() Node {
	return e.rChild
}
