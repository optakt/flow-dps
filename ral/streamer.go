package ral

import (
	"bytes"
	"fmt"

	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
)

// Streamer is a wrapper around the random access ledger core that allows
// streaming the data to the core data structure.
type Streamer struct {
	blocks   map[flow.Identifier]struct{}             // keeping track of known blocks
	children map[flow.Identifier]flow.Identifier      // keeping track of block children
	commits  map[flow.Identifier]flow.StateCommitment // keeping track of state commitment for blocks
	trie     *trie.MTrie                              // the current execution state trie
	cache    []Delta                                  // the cache of deltas since last block
	active   flow.Identifier                          // the next sealed block state we are looking for
	sentinel flow.StateCommitment                     // what state commitment we are looking for next
}

// NewStreamer creates a new streamer as a sink for data to the core random
// access ledger.
func NewStreamer(root flow.Identifier, commit flow.StateCommitment) (*Streamer, error) {
	trie, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
	if err != nil {
		return nil, fmt.Errorf("could not initialize trie")
	}
	s := &Streamer{
		blocks:   make(map[flow.Identifier]struct{}),
		children: make(map[flow.Identifier]flow.Identifier),
		commits:  make(map[flow.Identifier]flow.StateCommitment),
		trie:     trie,
		cache:    []Delta{},
		active:   root,
		sentinel: commit,
	}
	s.blocks[root] = struct{}{}
	s.commits[root] = commit
	return s, nil
}

// Block adds a block to the minimal finalized chain that the streamer keeps
// track of in order to consolidate updates per block.
func (s *Streamer) Block(parentID flow.Identifier, blockID flow.Identifier) error {
	_, ok := s.blocks[blockID]
	if ok {
		return fmt.Errorf("block already exists (%x)", blockID)
	}
	childID, ok := s.children[parentID]
	if ok {
		return fmt.Errorf("parent already has child (parent: %x, existing: %x, provided: %x)", parentID, childID, blockID)
	}
	s.blocks[blockID] = struct{}{}
	s.children[parentID] = blockID
	return nil
}

// Seal adds a block seal to the minimal sealed chain that the streamer keeps
// track of in order to validate updates against committed state.
func (s *Streamer) Seal(blockID flow.Identifier, commit flow.StateCommitment) error {
	_, ok := s.blocks[blockID]
	if !ok {
		return fmt.Errorf("could not find block (%x)", blockID)
	}
	existing, ok := s.commits[blockID]
	if ok {
		return fmt.Errorf("commit already exists (block: %x, existing: %x, provided: %x)", blockID, existing, commit)
	}
	s.commits[blockID] = commit
	return nil
}

// Delta adds an execution state delta to the streamer, that can be applied to
// the state trie when its root hash corresponds to the given commit.
func (s *Streamer) Delta(commit flow.StateCommitment, delta Delta) error {
	hash := s.trie.RootHash()
	if !bytes.Equal(commit, hash) {
		return fmt.Errorf("could not match root hash (%x != %x)", commit, hash)
	}
	trie, err := trie.NewTrieWithUpdatedRegisters(s.trie, delta.Paths(), delta.Payloads())
	if err != nil {
		return fmt.Errorf("could not update trie (%w)", err)
	}
	s.trie = trie
	s.cache = append(s.cache, delta)
	hash = s.trie.RootHash()
	if !bytes.Equal(hash, s.sentinel) {
		return nil
	}
	fmt.Printf("matched %x => %x\n", s.active, s.sentinel)
	// compound all deltas and store accordingly
	child, ok := s.children[s.active]
	if !ok {
		// TODO: this should be handled gracefully
		return fmt.Errorf("could not find child (%x)", s.active)
	}
	commit, ok = s.commits[child]
	if !ok {
		// TODO: this should be handled gracefully
		return fmt.Errorf("could not find commit (%x)", child)
	}
	s.active = child
	s.sentinel = commit
	return nil
}
