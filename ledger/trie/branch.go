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

	hash   hash.Hash
	height int
}

func NewBranch(height int) *Branch {
	n := Branch{
		height: height,
	}

	return &n
}

func (b Branch) Hash() hash.Hash {
	var lHash, rHash hash.Hash
	if b.lChild != nil {
		lHash = b.lChild.Hash()
	} else {
		lHash = ledger.GetDefaultHashForHeight(b.height - 1)
	}

	if b.rChild != nil {
		rHash = b.rChild.Hash()
	} else {
		rHash = ledger.GetDefaultHashForHeight(b.height - 1)
	}

	return hash.HashInterNode(lHash, rHash)
}

func (b Branch) Dump(w io.Writer) {
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