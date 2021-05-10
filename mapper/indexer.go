package mapper

import (
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/model"
)

type Indexer interface {
	Index(height uint64, blockID flow.Identifier, commit flow.StateCommitment, deltas []model.Delta) error
}
