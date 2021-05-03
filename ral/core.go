package ral

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
)

// Core represents the core indexing infrastructure of the random access ledger.
type Core struct {
	log   zerolog.Logger
	index *badger.DB
	trie  *trie.MTrie // the current execution state trie
}

// NewCore creates a new random access ledger core, using the provided badger
// database as a backend.
// WARNING: this should be a separate database from the protocol state!
func NewCore(log zerolog.Logger, index *badger.DB) (*Core, error) {
	trie, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
	if err != nil {
		return nil, fmt.Errorf("could not initialize empty trie (%w)", err)
	}
	c := &Core{
		log:   log.With().Str("component", "core").Logger(),
		index: index,
		trie:  trie,
	}
	return c, nil
}

// Index is used to index a new set of state deltas for the given block.
func (c *Core) Index(height uint64, blockID flow.Identifier, commit flow.StateCommitment, deltas []Delta) error {

	c.log.Info().
		Uint64("height", height).
		Hex("block", blockID[:]).
		Hex("commit", commit[:]).
		Int("deltas", len(deltas)).
		Msg("indexing state deltas")

	// let's use a single transaction to make indexing of a new block atomic
	tx := c.index.NewTransaction(true)

	// first, map the block ID to the height for easy lookup later
	key := make([]byte, len(blockID)+1)
	key[0] = BlockToHeight
	copy(key[1:], blockID[:])
	val := make([]byte, 8)
	binary.BigEndian.PutUint64(val, height)
	err := tx.Set(key, val)
	if err != nil {
		return fmt.Errorf("could not set block-to-height index (%w)", err)
	}

	// second, map the commit to the height for easy lookup later
	key = make([]byte, len(commit)+1)
	key[0] = CommitToHeight
	copy(key[1:], commit)
	err = tx.Set(key, val)
	if err != nil {
		return fmt.Errorf("could not set commit-to-height index (%w)", err)
	}

	// finally, we index the payload for every path that has changed in this block
	for _, delta := range deltas {
		for _, change := range delta {
			key = make([]byte, pathfinder.PathByteSize+8)
			copy(key[:pathfinder.PathByteSize], change.Path)
			binary.BigEndian.PutUint64(key[pathfinder.PathByteSize:], height)
			// TODO: update to capnproto encoding for performance
			val, err := json.Marshal(change.Payload)
			if err != nil {
				return fmt.Errorf("could not encode payload (%w)", err)
			}
			err = tx.Set(key, val)
			if err != nil {
				return fmt.Errorf("could not set path payload (%w)", err)
			}
		}
	}

	// let's not forget to finalize the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("could not commit transaction (%w)", err)
	}

	return nil
}

// Height returns the first height for a given state commitment.
func (c *Core) Height(commit flow.StateCommitment) (uint64, error) {

	// build the key and look up the height for the commit
	key := make([]byte, len(commit)+1)
	key[0] = CommitToHeight
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
		return 0, fmt.Errorf("could not get commit-to-height index (%w)", err)
	}

	return height, nil
}

// Payload returns the payload of the given path at the given block height.
func (c *Core) Payload(height uint64, path ledger.Path) (*ledger.Payload, error) {

	// Use seek on Badger to seek to the next biggest key lower than the key we
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
			return fmt.Errorf("no payload for register found")
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

// Ledger allows us to access the ledger API at a snapshot with the specified
// option parameters.
func (c *Core) Ledger(options ...func(*Ledger)) *Ledger {
	ledger := &Ledger{
		core:    c,
		version: complete.DefaultPathFinderVersion,
	}
	for _, option := range options {
		option(ledger)
	}
	return ledger
}
