package dps

import (
	"github.com/onflow/flow-go/model/flow"
)

type Indexer interface {
	Index(height uint64, blockID flow.Identifier, commit flow.StateCommitment, deltas []Delta) error
}
