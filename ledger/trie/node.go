package trie

import (
	"io"

	"github.com/onflow/flow-go/ledger/common/hash"
)

type Node interface{
	Hash() hash.Hash
	Dump(io.Writer)
}