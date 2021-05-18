package state

import (
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/service/storage"
)

type Commit struct {
	core *Core
}

func (c *Commit) ForHeight(height uint64) (flow.StateCommitment, error) {
	var commit flow.StateCommitment
	err := c.core.db.View(storage.RetrieveCommitByHeight(height, &commit))
	return commit, err
}
