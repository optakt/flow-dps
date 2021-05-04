package dps

import (
	"github.com/onflow/flow-go-sdk"
)

type Chain interface {
	Next() error
	Height() uint64
	Block() flow.Identifier
	Commit() []byte
}
