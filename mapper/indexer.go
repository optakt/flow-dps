package mapper

import (
	"github.com/awfm9/flow-dps/model"
	"github.com/onflow/flow-go/model/flow"
)

type Indexer interface {
	Index(height uint64, blockID flow.Identifier, commit flow.StateCommitment, deltas []model.Delta) error
}
