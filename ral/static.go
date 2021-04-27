package ral

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"
)

type Active struct {
	Height  uint64
	BlockID flow.Identifier
	Commit  flow.StateCommitment
}

// Static is a random access ledger that bootstraps the index from a static
// snapshot of the protocol state and the corresponding ledger write-ahead log.
type Static struct {
	db     *badger.DB
	trie   *trie.MTrie // the current execution state trie
	active *Active     // the block we currently try to match
	cache  []Delta
}

// NewStatic creates a new random access ledger, bootstrapping the state from
// the provided badger snapshot and write-ahead log.
func NewStatic(db *badger.DB) (*Static, error) {
	var height uint64
	err := operation.RetrieveRootHeight(&height)(db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve root height (%w)", err)
	}
	var rootID flow.Identifier
	err = operation.LookupBlockHeight(height, &rootID)(db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not look up block height (%w)", err)
	}
	var sealID flow.Identifier
	err = operation.LookupBlockSeal(rootID, &sealID)(db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not look up block seal (%w)", err)
	}
	var seal flow.Seal
	err = operation.RetrieveSeal(sealID, &seal)(db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve seal (%w)", err)
	}
	trie, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
	if err != nil {
		return nil, fmt.Errorf("could not initialize trie")
	}
	active := &Active{
		Height:  height,
		BlockID: rootID,
		Commit:  seal.FinalState,
	}
	s := &Static{
		db:     db,
		trie:   trie,
		active: active,
		cache:  []Delta{},
	}
	return s, nil
}

// Delta adds an execution state delta to the streamer, that can be applied to
// the state trie when its root hash corresponds to the given commit.
func (s *Static) Delta(commit flow.StateCommitment, delta Delta) error {

	// We first do a sanity check to make sure that the provided delta is
	// supposed to be applied to a state trie with the given root hash.
	hash := s.trie.RootHash()
	if !bytes.Equal(commit, hash) {
		return fmt.Errorf("could not match root hash (%x != %x)", commit, hash)
	}

	// Next, we apply the provided delta to the trie and cache the delta until
	// we found the next state commitment we are looking for.
	trie, err := trie.NewTrieWithUpdatedRegisters(s.trie, delta.Paths(), delta.Payloads())
	if err != nil {
		return fmt.Errorf("could not update trie (%w)", err)
	}
	s.cache = append(s.cache, delta)
	s.trie = trie

	// If we have reached a the state commitment of the currently active block,
	// we can map the deltas accordingly and forward to the next one, until we
	// find a block that expects a different state commitment.
	for {

		// check if the current trie root hash corresponds to the state
		// commitment of the currently active block that we are looking for
		hash = s.trie.RootHash()
		if !bytes.Equal(hash, s.active.Commit) {
			break
		}

		fmt.Printf("matched: %x => %x (%d updates)\n", s.active.BlockID, s.active.Commit, len(s.cache))

		// if the trie root hash does indeed correspond to the state committed
		// at the currently active block, we can store the deltas in the cache
		// as the delta between the last block and the active one
		// TODO: store deltas
		s.cache = []Delta{}

		// then we can forward the currently active block to the next one in the
		// chain; if no more blocks are there, we break
		var blockID flow.Identifier
		err = operation.LookupBlockHeight(s.active.Height, &blockID)(s.db.NewTransaction(false))
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("could not look up block height (%w)", err)
		}
		var sealID flow.Identifier
		err = operation.LookupBlockSeal(blockID, &sealID)(s.db.NewTransaction(false))
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("could not look up block seal (%w)", err)
		}
		var seal flow.Seal
		err = operation.RetrieveSeal(sealID, &seal)(s.db.NewTransaction(false))
		if err != nil {
			return fmt.Errorf("could not retrieve seal (%w)", err)
		}
		s.active = &Active{
			Height:  s.active.Height + 1,
			BlockID: blockID,
			Commit:  seal.FinalState,
		}
	}

	return nil
}
