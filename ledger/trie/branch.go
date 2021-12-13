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
	"fmt"
	"io"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type Branch struct {
	lChild Node
	rChild Node

	height uint16
	hash   hash.Hash

	// dirty marks whether the current hash value of the branch is valid.
	// If this is set to false, the hash needs to be recomputed.
	dirty bool
}

func NewBranch(height uint16, lChild, rChild Node) *Branch {
	b := Branch{
		lChild: lChild,
		rChild: rChild,

		height: height,
		dirty:  true,
	}

	return &b
}

func NewBranchWithHash(height uint16, hash hash.Hash, lChild, rChild Node) *Branch {
	b := Branch{
		lChild: lChild,
		rChild: rChild,

		height: height,
		hash:   hash,
		dirty:  false,
	}

	return &b
}

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

func (b *Branch) Hash() hash.Hash {
	if b.dirty {
		b.computeHash()
	}
	return b.hash
}

func (b *Branch) FlagDirty() {
	b.dirty = true
}

func (b *Branch) Height() uint16 {
	return b.height
}

func (b *Branch) Path() ledger.Path {
	return ledger.DummyPath
}

func (b *Branch) LeftChild() Node {
	return b.lChild
}

func (b *Branch) RightChild() Node {
	return b.rChild
}

func (b *Branch) Dump(w io.Writer) {
	_, err := w.Write([]byte(fmt.Sprintf("%d:\tBRANCH\t%x\n", b.height, b.Hash())))
	if err != nil {
		panic(err)
	}

	if b.lChild != nil {
		b.lChild.Dump(w)
	}
	if b.rChild != nil {
		b.rChild.Dump(w)
	}
}
