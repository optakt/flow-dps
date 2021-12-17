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

// Leaf is what contains the values in the trie. This implementation uses nodes that are
// compacted and do not always reside at the bottom layer of the trie.
// Instead, they are inserted at the first heights where they do not conflict with others.
// This allows the trie to keep a relatively small amount of nodes, instead of having
// many nodes/extensions for each leaf in order to bring it all the way to the bottom
// of the trie.
type Leaf struct {
	path   ledger.Path
	hash   hash.Hash
	height uint16
}

// NewLeaf creates a new leaf at the given height, and computes its hash using the
// given path and payload.
func NewLeaf(height uint16, path ledger.Path, payload *ledger.Payload) *Leaf {
	n := Leaf{
		path:   path,
		hash:   ledger.ComputeCompactValue(hash.Hash(path), payload.Value, int(height)),
		height: height,
	}

	return &n
}

// Hash returns the leaf hash.
func (l Leaf) Hash() hash.Hash {
	return l.hash
}

// Height returns the extension height.
func (l Leaf) Height() uint16 {
	return l.height
}

// Path returns the leaf path.
func (l Leaf) Path() ledger.Path {
	return l.path
}

// LeftChild returns nothing since leaves do not have any children.
func (l Leaf) LeftChild() Node {
	return nil
}

// RightChild returns nothing since leaves do not have any children.
func (l Leaf) RightChild() Node {
	return nil
}
