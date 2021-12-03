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
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type Extension struct {
	lChild Node
	rChild Node

	height uint16
	// skip is the height up to which the extension skips to.
	skip uint16
	// path is the path that the extension skips through.
	// FIXME: Store only skipped part of the path rather than whole path.
	path ledger.Path
	hash hash.Hash

	// dirty marks whether the current hash value of the branch is valid.
	// If this is set to false, the hash needs to be recomputed.
	dirty bool
}

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

func NewExtensionWithHash(height, skip uint16, path ledger.Path, hash hash.Hash, lChild, rChild Node) *Extension {
	e := Extension{
		lChild: lChild,
		rChild: rChild,

		height: height,
		skip:   skip,
		path:   path,

		hash:  hash,
		dirty: false,
	}

	return &e
}

func (e *Extension) computeHash() {
	var computed hash.Hash

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

func (e *Extension) Hash() hash.Hash {
	if e.dirty {
		e.computeHash()
	}
	return e.hash
}

func (e *Extension) FlagDirty() {
	e.dirty = true
}

func (e *Extension) Height() uint16 {
	return e.height
}

func (e *Extension) LeftChild() Node {
	return e.lChild
}

func (e *Extension) RightChild() Node {
	return e.rChild
}

func (e *Extension) Path() ledger.Path {
	return ledger.DummyPath
}

func (e *Extension) Dump(w io.Writer) {
	_, err := w.Write([]byte(fmt.Sprintf("%d:\tEXTENS\t%d\t%x\n", e.height, e.skip, e.Hash())))
	if err != nil {
		panic(err)
	}

	if e.lChild != nil {
		e.lChild.Dump(w)
	}
	if e.rChild != nil {
		e.rChild.Dump(w)
	}
}
