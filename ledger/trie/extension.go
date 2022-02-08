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
	hash  [32]byte
	dirty bool
	path  []byte
	count uint8
	child Node
}

// Hash returns the extension hash. If it is currently dirty, it is recomputed first.
func (e *Extension) Hash(height uint8, path [32]byte, getPayload payloadRetriever) [32]byte {
	if e.dirty {
		e.computeHash(height, path, getPayload)
	}
	return e.hash
}

// computeHash computes the extension's hash.
func (e *Extension) computeHash(height uint8, _ [32]byte, getPayload payloadRetriever) {
	defer func() {
		e.dirty = false
	}()

	// Build the path for the child, based on the parent of the extension.
	var childPath [32]byte
	// For each skipped height, set the bits in the child path accordingly.
	// FIXME: Extension writes one bit too much in the child path. If this change does not work
	//  we can also unset the bit later maybe?
	depth := ledger.NodeMaxHeight - 1 - height
	for i := 0; i <= int(depth)+int(e.count); i++ {
		if bitutils.Bit(e.path[:], i) == 1 {
			bitutils.SetBit(childPath[:], i)
		}
	}

	// If the child is a leaf, simply use its hash as the extension's hash,
	// since in that case the extension is the equivalent of a Flow "compact leaf".
	// The leaf needs to use the height of its extension for hash computation.
	leaf, ok := e.child.(*Leaf)
	if ok {
		leaf.dirty = true
		e.hash = e.child.Hash(height, childPath, getPayload)
		return
	}

	// If the child is not a leaf, the height it needs for hash computation
	// is the height at the bottom of the extension.
	h := e.child.Hash(height-e.count, childPath, getPayload)

	// For each skipped height, combine the previous hash with the default ledger
	// height of the current layer.
	var lHash, rHash hash.Hash
	for i := int(height) - int(e.count) + 1; i <= int(height); i++ {
		if bitutils.Bit(e.path[:], 255-i) == 0 {
			lHash = h
			rHash = ledger.GetDefaultHashForHeight(i)
		} else {
			lHash = ledger.GetDefaultHashForHeight(i)
			rHash = h
		}
		h = hash.HashInterNode(lHash, rHash)
	}

	e.hash = h
	return
}
