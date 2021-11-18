package trie

import (
	"io"

	"github.com/onflow/flow-go/ledger/common/hash"
)

type ParentNode interface {
	Node

	SetChildren(lChild, rChild Node)
}

type Node interface {
	Hash() hash.Hash
	ComputeHash() hash.Hash
	Dump(io.Writer)
}
