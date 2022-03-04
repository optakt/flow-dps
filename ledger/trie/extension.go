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
	trie "github.com/onflow/flow-go/ledger/common/hash"
)

// Extension nodes are what turn the state trie into a sparse trie. They hold a
// number of bits of the path, where all nodes in that part of the trie share these
// same bits of the path. They drastically reduce the memory needed for the trie
// when most of the paths on the trie are not populated.
type Extension struct {

	// Extension nodes can be split when inserting additional nodes into sparse
	// parts of the trie. In those cases, their hash becomes invalid and needs
	// to be recomputed. In order to avoid redundant recomputations upon multiple
	// insertions, we lazily recompute when the hash of the trie is requested.
	hash  [32]byte
	clean bool

	// The full path of the insertion will be stored in the leaf node anyway, so
	// we can simplify a lot of code by keeping the whole path the extension is
	// on. We can then use the current depth as starting point for the extension's
	// bits, and the count as the cut-off at the end.
	path  *ledger.Path
	count uint8

	// An extension can either have a branch as a child, in case it doesn't reach
	// the bottom of the trie, or a leaf, in case it extends all the way to the
	// bottom.
	child Node
}

// Hash returns the extension hash. If it is currently dirty, it is recomputed first.
func (e *Extension) Hash(height int) hash.Hash {
	if !e.clean {
		e.hash = e.computeHash(height)
		e.clean = true
	}
	return e.hash
}

// computeHash computes the extension's hash.
func (e *Extension) computeHash(height int) hash.Hash {

	// If the child is a leaf, simply use its hash as the extension's hash,
	// since in that case the extension is the equivalent of a Flow "compact leaf".
	_, ok := e.child.(*Leaf)
	if ok {
		hash := e.child.Hash(height)
		return hash
	}

	// If the child is not a leaf, we use its hash as the starting point for
	// the extension's hash. We then hash it against the default hash for each
	// height for every bit on the extension.
	hash := e.child.Hash(height - int(e.count) - 1)
	for i := height - int(e.count); i <= height; i++ {
		empty := ledger.GetDefaultHashForHeight(i)
		if bitutils.Bit(e.path[:], 255-i) == 0 {
			hash = trie.HashInterNode(hash, empty)
		} else {
			hash = trie.HashInterNode(empty, hash)
		}
	}

	return hash
}
