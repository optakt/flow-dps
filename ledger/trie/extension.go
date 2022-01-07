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
func (e *Extension) Hash() [32]byte {
	if e.dirty {
		e.computeHash()
	}
	return e.hash
}

// computeHash computes the extension's hash.
func (e *Extension) computeHash() {
	computed := e.child.Hash()
	for i := 0; i < int(e.skip); i++ {
		if bitutils.Bit(e.path[:], int(nodeHeight(i+1))) == 0 {
			computed = hash.HashInterNode(computed, ledger.GetDefaultHashForHeight(int(i)))
		} else {
			computed = hash.HashInterNode(ledger.GetDefaultHashForHeight(int(i)), computed)
		}
	}
	e.hash = computed
	e.dirty = false
}
