package ral

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
)

// Core represents the core indexing infrastructure of the random access ledger.
type Core struct {
	index *badger.DB
	trie  *trie.MTrie // the current execution state trie
	tip   uint64
}

// NewCore creates a new random access ledger core, using the provided badger
// database as a backend.
// WARNING: this should be a separate database from the protocol state!
func NewCore(index *badger.DB) (*Core, error) {
	trie, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
	if err != nil {
		return nil, fmt.Errorf("could not initialize empty trie (%w)", err)
	}
	c := &Core{
		index: index,
		trie:  trie,
		tip:   0,
	}
	return c, nil
}

// Index is used to index a new set of state deltas for the given block.
func (c *Core) Index(height uint64, blockID flow.Identifier, deltas []Delta) error {
	fmt.Printf("indexing block deltas: %d - %x - %d deltas\n", height, blockID, len(deltas))
	for index, delta := range deltas {
		fmt.Printf("indexing delta %d: %d changes\n", index, len(delta))
		for _, change := range delta {
			key := make([]byte, pathfinder.PathByteSize+8)
			copy(key[:pathfinder.PathByteSize], change.Path)
			binary.BigEndian.PutUint64(key[pathfinder.PathByteSize:], height)
			// TODO: update to capnproto encoding for performance
			val, err := json.Marshal(change.Payload)
			if err != nil {
				return fmt.Errorf("could not encode payload (%w)", err)
			}
			err = c.index.Update(func(tx *badger.Txn) error {
				return tx.Set(key, val)
			})
			if err != nil {
				return fmt.Errorf("could not update database (%w)", err)
			}
		}
	}
	// TODO: make sure this is properly managed for bootstrapping and subsequent
	// updates with sanity check
	c.tip = height
	return nil
}

// Payload returns the payload of the given path at the given block height.
func (c *Core) Payload(height uint64, path ledger.Path) (*ledger.Payload, error) {

	// if the height is beyond tip, we can't answer the query
	if height > c.tip {
		return nil, fmt.Errorf("unknown block height (%d)", height)
	}

	// Use seek on Badger to seek to the next biggest key lower than the key we
	// seek for; this should represent the last update to the path before the
	// requested height and should thus be the payload we care about.
	var payload *ledger.Payload
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
			return fmt.Errorf("no payload for register found")
		}
		err := it.Item().Value(func(val []byte) error {
			err := json.Unmarshal(val, payload)
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

	return nil, nil
}

// Ledger allows us to access the ledger API at a snapshot with the specified
// option parameters.
// TODO: separate this into multiple API wrappers; the ledger contains a state
// hash that we have to use to get the height, so it's pointless to provide the
// height option.
func (c *Core) Ledger(options ...func(*Snapshot)) *Snapshot {
	snapshot := &Snapshot{
		core:    c,
		version: complete.DefaultPathFinderVersion,
		height:  c.tip,
	}
	for _, option := range options {
		option(snapshot)
	}
	return snapshot
}
