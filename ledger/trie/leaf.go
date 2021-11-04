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
	height int
}

func NewLeaf(path ledger.Path, payload *ledger.Payload, height int) *Leaf {
	n := Leaf{
		path:   path,
		hash:   ledger.ComputeCompactValue(hash.Hash(path), payload.Value, height),
		height: height,
	}

	return &n
}

func (l Leaf) Hash() hash.Hash {
	return l.hash
}

func (l Leaf) Dump(w io.Writer) {
	_, err := w.Write([]byte(fmt.Sprintf("%d:\tLEAF\t%x\t%x\n", l.height, l.hash, l.path[:])))
	if err != nil {
		panic(err)
	}
}
