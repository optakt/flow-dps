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
	"github.com/onflow/flow-go/ledger/common/hash"
)

// Branch nodes are the non-compact default nodes of a trie. They point to the
// two possible branches of the trie below them, one for each bit. In the sparse
// implementation of the trie, they are replaced with extension nodes whenever
// one of their children would be empty.
type Branch struct {

	// The hash of a default node is a hash of both of its children. If either
	// of the children change, such as when extension nodes are modified, the
	// hash needs to be recomputed. We do this in a lazy manner when the trie
	// hash is requested, so we can avoid redundant hash computations when doing
	// multiple insertions.
	hash  hash.Hash
	clean bool

	// The left node of a branch points to the node where the path continues with
	// a `0` bit; the right node points to the `1` bit.
	left  Node
	right Node
}

// Hash returns the branch hash. If it is currently dirty, it is recomputed first.
func (b *Branch) Hash(height int) hash.Hash {
	if !b.clean {
		b.hash = b.computeHash(height)
		b.clean = true
	}
	return b.hash
}

// computeHash computes the branch hash by hashing its children.
func (b *Branch) computeHash(height int) hash.Hash {
	left := b.left.Hash(height - 1)
	right := b.right.Hash(height - 1)
	hash := hash.HashInterNode(left, right)
	return hash
}
