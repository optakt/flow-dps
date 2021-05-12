package state

import (
	"encoding/binary"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/onflow/flow-go/model/flow"
)

type Commit struct {
	core *Core
}

func (c *Commit) ForHeight(height uint64) (flow.StateCommitment, error) {
	key := make([]byte, 1+8)
	key[0] = prefixBlockIndex
	binary.BigEndian.PutUint64(key[1:1+8], height)
	var commit flow.StateCommitment
	err := c.core.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return fmt.Errorf("could not retrieve height index: %w", err)
		}
		_ = item.Value(func(val []byte) error {
			copy(commit, val)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not look up height: %w", err)
	}
	return commit, nil
}
