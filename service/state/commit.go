package state

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"
)

type Commit struct {
	core *Core
}

func (c *Commit) ForHeight(height uint64) (flow.StateCommitment, error) {
	var commit flow.StateCommitment
	err := c.core.db.View(func(tx *badger.Txn) error {
		return RetrieveCommit(height, &commit)(tx)
	})
	if err != nil {
		return nil, fmt.Errorf("could not look up height: %w", err)
	}
	return commit, nil
}
