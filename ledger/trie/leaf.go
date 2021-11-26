package trie

import (
	"fmt"
	"io"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type Leaf struct {
	path   ledger.Path
	hash   hash.Hash
	height uint16
}

func NewLeaf(path ledger.Path, payload *ledger.Payload, height uint16) *Leaf {
	n := Leaf{
		path:   path,
		hash:   ledger.ComputeCompactValue(hash.Hash(path), payload.Value, int(height)),
		height: height,
	}

	return &n
}

func (l Leaf) Hash() hash.Hash {
	return l.hash
}

func (l Leaf) Height() uint16 {
	return l.height
}

func (l Leaf) Path() ledger.Path {
	return l.path
}

func (l Leaf) LeftChild() Node {
	return nil
}

func (l Leaf) RightChild() Node {
	return nil
}

func (l Leaf) Dump(w io.Writer) {
	_, err := w.Write([]byte(fmt.Sprintf("%d:\tLEAF\t%x\t%x\n", l.height, l.hash, l.path[:])))
	if err != nil {
		panic(err)
	}
}
