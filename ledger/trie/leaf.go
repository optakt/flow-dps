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
	"golang.org/x/sync/semaphore"
)

// Leaf nodes are found at the end of each path of the trie. They do not contain
// a part of the path, so they could, in theory, be shuffled around easily. This
// is made more difficult by the Flow implementation of the trie, which hashes
// the height of a leaf as part of the node hash, and uses the a height based on
// the sparse trie, which changes as the trie fills up, instead of the height in
// terms of path traversed, which would always be the same.
type Leaf struct {

	// The hash of the leaf.
	hash  hash.Hash
	clean bool

	// The path of the laf.
	path *ledger.Path

	// The payload of the leaf.
	payload *ledger.Payload
}

// Hash returns the leaf hash.
func (l *Leaf) Hash(_ *semaphore.Weighted, height int) hash.Hash {
	if !l.clean {
		l.hash = l.computeHash(height)
		l.clean = true
	}
	return l.hash
}

func (l *Leaf) computeHash(height int) hash.Hash {
	return ledger.ComputeCompactValue(hash.Hash(*l.path), l.payload.Value, height)
}
