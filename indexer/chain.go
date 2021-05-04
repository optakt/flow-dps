package indexer

import (
	"github.com/onflow/flow-go/model/flow"
)

type Step struct {
	BlockID flow.Identifier
	Height  uint64
	Commit  flow.StateCommitment
}

type Chain interface {
	Next() bool
	Step() Step
}
