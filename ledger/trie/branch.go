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

func (b *Branch) computeHash() {
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