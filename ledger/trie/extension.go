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

	skip   int // Height at which the extension points to.
	path   ledger.Path // FIXME: Store only skipped part of the path rather than whole path.
	hash   hash.Hash
	height int
}

func NewExtension(height int, skip int, path ledger.Path) *Extension {
	e := Extension{
		skip:   skip,
		path:   path,
		height: height,
	}

	//fmt.Printf("Creating extension that goes from height %d to %d for path %x\n", height, skip, path[:])

	return &e
}

func (e *Extension) ComputeHash() hash.Hash {
	var computed hash.Hash

	var lHash, rHash hash.Hash
	if e.lChild != nil {
		lHash = e.lChild.ComputeHash()
	} else {
		lHash = ledger.GetDefaultHashForHeight(e.skip-1)
	}

	if e.rChild != nil {
		rHash = e.rChild.ComputeHash()
	} else {
		rHash = ledger.GetDefaultHashForHeight(e.skip-1)
	}
	computed = hash.HashInterNode(lHash, rHash)
	//fmt.Printf("Computing hash for extension at path")
	//for _, p1 := range e.path[:] {
	//	fmt.Printf("%08b", p1)
	//}
	//fmt.Println()
	//fmt.Printf("GOT H%d %x + %x = %x\n", e.skip-1, lHash[:], rHash[:], computed[:])

	for i := e.skip; i < e.height; i++ {
		if bitutils.Bit(e.path[:], nodeHeight(i+1)) == 0 {
			lHash = computed
			rHash = ledger.GetDefaultHashForHeight(i)
		} else {
			lHash = ledger.GetDefaultHashForHeight(i)
			rHash = computed
		}
		computed = hash.HashInterNode(lHash, rHash)
		//fmt.Printf("GOT H%d %x + %x = %x\n", i, lHash[:], rHash[:], computed[:])
	}

	e.hash = computed

	return e.hash
}

func (e *Extension) Hash() hash.Hash {
	return e.ComputeHash()
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

func (e *Extension) SetChildren(lChild, rChild Node) {
	e.lChild = lChild
	e.rChild = rChild
}
