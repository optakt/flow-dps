package ledger

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/awfm9/flow-dps/model"
	"github.com/dgraph-io/badger/v2"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/model/flow"
)

type Core struct {
	index *badger.DB
}

func NewCore(index *badger.DB) (*Core, error) {
	c := &Core{
		index: index,
	}
	return c, nil
}

// Height returns the first height for a given state commitment.
func (c *Core) Height(commit flow.StateCommitment) (uint64, error) {

	// build the key and look up the height for the commit
	key := make([]byte, len(commit)+1)
	key[0] = model.CommitToHeight
	copy(key[1:], commit)
	var height uint64
	err := c.index.View(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		_ = item.Value(func(val []byte) error {
			height = binary.BigEndian.Uint64(val)
			return nil
		})
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not look up height for commit (%w)", err)
	}

	return height, nil
}

// Payload returns the payload of the given path at the given block height.
func (c *Core) Payload(height uint64, path ledger.Path) (*ledger.Payload, error) {

	// Use seek on Ledger to seek to the next biggest key lower than the key we
	// seek for; this should represent the last update to the path before the
	// requested height and should thus be the payload we care about.
	var payload ledger.Payload
	key := make([]byte, pathfinder.PathByteSize+8)
	copy(key[:pathfinder.PathByteSize], path)
	binary.BigEndian.PutUint64(key[pathfinder.PathByteSize:], height)
	err := c.index.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.IteratorOptions{
			PrefetchSize:   0,
			PrefetchValues: false,
			Reverse:        true,
			AllVersions:    false,
			InternalAccess: false,
			Prefix:         path,
		})
		it.Seek(key)
		if !it.Valid() {
			return fmt.Errorf("could not find register for path")
		}
		err := it.Item().Value(func(val []byte) error {
			err := json.Unmarshal(val, &payload)
			if err != nil {
				return fmt.Errorf("could not decode payload (%w)", err)
			}
			return nil
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve payload (%w)", err)
	}

	return &payload, nil
}

func (c *Core) Raw(options ...func(*Raw)) *Raw {
	r := Raw{}
	return &r
}

func (c *Core) Ledger(options ...func(*Ledger)) *Ledger {
	l := Ledger{}
	return &l
}
