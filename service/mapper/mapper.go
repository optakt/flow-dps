// Copyright 2021 Alvalor S.A.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package mapper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"

	"github.com/awfm9/flow-dps/models/dps"
)

type Mapper struct {
	log    zerolog.Logger
	chain  Chain
	feed   Feeder
	index  dps.Index
	tree   *trie.MTrie
	height uint64
	wg     *sync.WaitGroup
	stop   chan struct{}
}

// New creates a new mapper that uses chain data to map trie updates to blocks
// and then passes on the details to the indexer for indexing.
func New(log zerolog.Logger, chain Chain, feed Feeder, index dps.Index, options ...func(*MapperConfig)) (*Mapper, error) {

	// We don't use a checkpoint by default. The options can set one, in which
	// case we will start from the checkpoint instead of an empty trie.
	cfg := MapperConfig{
		CheckpointFile: "",
	}
	for _, option := range options {
		option(&cfg)
	}

	// Get the root height so we know where to start at.
	height, err := chain.Root()
	if err != nil {
		return nil, fmt.Errorf("could not get root height: %w", err)
	}

	// If we have a checkpoint file, it should be a root checkpoint, so it
	// should only contain a single trie that we load as our initial root state.
	// Otherwise, the root state is an empty memory trie.
	tree, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
	if err != nil {
		return nil, fmt.Errorf("could not initialize empty memory trie: %w", err)
	}
	if cfg.CheckpointFile != "" {
		file, err := os.Open(cfg.CheckpointFile)
		if err != nil {
			return nil, fmt.Errorf("could not open checkpoint file: %w", err)
		}
		checkpoint, err := wal.ReadCheckpoint(file)
		if err != nil {
			return nil, fmt.Errorf("could not read checkpoint: %w", err)
		}
		tries, err := flattener.RebuildTries(checkpoint)
		if err != nil {
			return nil, fmt.Errorf("could not rebuild tries: %w", err)
		}
		if len(tries) != 1 {
			return nil, fmt.Errorf("should only have one trie in root checkpoint (tries: %d)", len(tries))
		}
		tree = tries[0]
	}

	// NOTE: there might be a number of trie updates in the WAL before the root
	// block, which means that we can not sanity check the state trie against
	// the root block state commitment here.

	i := Mapper{
		log:    log,
		chain:  chain,
		feed:   feed,
		index:  index,
		tree:   tree,
		height: height,
		wg:     &sync.WaitGroup{},
		stop:   make(chan struct{}),
	}

	return &i, nil
}

func (m *Mapper) Stop(ctx context.Context) error {
	close(m.stop)
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (m *Mapper) Run() error {
	m.wg.Add(1)
	defer m.wg.Done()

	// The purpose of this function is to map state deltas from a continuous
	// feed to specific blocks from the chain. This is necessary because the
	// trie updates that we receive as state deltas are agnostic of blocks and
	// instead operate on a chunk level. This means that we will run into the
	// state commitment of every finalized block in the chain, as long as we
	// keep applying state deltas to the state trie and checking the root hash
	// of the state trie against the state commitment of the next block in the
	// chain.
	// This is what we do with these two loops. The outer loop skips over the
	// inner loop each time that the root hash of the state trie does *not*
	// match the state commitment of the next block in the state. It then
	// proceeds to retrieving the next state delta and applying it to the state
	// trie, which will be compared against the state commitment of the next
	// block in the chain again on the next iteration.
	// Once the root hash of the state trie matches the state commitment of the
	// next block in the chain, we go into the inner loop. In the inner loop,
	// we index the next block with its state commitment and its state deltas.
	// Every subsequent block is then also matched, which is why we have the
	// inner loop, as long as the state commitment doesn't change. As soon as a
	// new state commitment shows up on the chain, we go back to iterating in
	// the outer loop until we have assembled the necessary deltas to match the
	// new state commitment again.
	height := m.height
	tree := m.tree
	deltas := make([]dps.Delta, 0, 16)
Outer:
	for {

		// The second loop is responsible for mapping the currently active block
		// to the set of deltas that were collected. If the state commitment for
		// the block we are looking for isn't the same as the trie root hash, we
		// will immediately go to the next iteration of the outer loop to keep
		// adding deltas to the trie. If it does match, we will index the block
		// with the set of deltas we collected. This might happen more than once
		// if no change to the state trie happens between multiple blocks, at
		// which point we map the second and any subsequent blocks without
		// change to an empty set of deltas.

		commitTree := tree.RootHash()
		var log zerolog.Logger
	Inner:
		for {

			log = m.log.With().
				Uint64("height", height).
				Hex("commit_trie", commitTree).
				Int("num_deltas", len(deltas)).
				Logger()

			// We first try to get the next commit by height, because that is
			// the sign that the block has been sealed. If the retrieval times
			// out, we loop right back into this condition, because it means the
			// network might be stalling. If the error indicates we finished,
			// then we reached the end of the WAL and can finish without error.
			commitHeight, err := m.chain.Commit(height)
			if errors.Is(err, dps.ErrTimeout) {
				log.Warn().Msg("commit retrieval timed out, retrying")
				continue Inner
			}
			if errors.Is(err, dps.ErrFinished) {
				log.Debug().Msg("end of commit chain reached, stopping")
				break Outer
			}
			if err != nil {
				return fmt.Errorf("commit retrieval failed: %w", err)
			}

			log = log.With().Hex("commit_height", commitHeight).Logger()

			if !bytes.Equal(commitTree, commitHeight) {
				log.Debug().Msg("commit between tree and height does not match, keep searching")
				break Inner
			}

			header, err := m.chain.Header(height)
			if err != nil {
				return fmt.Errorf("could not retrieve header: %w (height: %d)", err, height)
			}

			blockID := header.ID()
			log = log.With().Hex("block", blockID[:]).Logger()

			events, err := m.chain.Events(height)
			if err != nil {
				return fmt.Errorf("could not retrieve events: %w (height: %d)", err, height)
			}

			log = log.With().Int("num_events", len(events)).Logger()

			// TODO: look at performance of doing separate transactions versus
			// having an API that allows combining into a single Badger tx
			// => https://github.com/awfm9/flow-dps/issues/36

			// If we successfully retrieved the commit, we can index everything
			// for this block, because everything should be available.
			err = m.index.Header(height, header)
			if err != nil {
				return fmt.Errorf("could not index header: %w", err)
			}
			err = m.index.Commit(height, commitHeight)
			if err != nil {
				return fmt.Errorf("could not index commit: %w", err)
			}
			err = m.index.Deltas(height, deltas)
			if err != nil {
				return fmt.Errorf("could not index deltas: %w", err)
			}
			err = m.index.Events(height, events)
			if err != nil {
				return fmt.Errorf("could not index events: %w", err)
			}
			err = m.index.Last(commitHeight)
			if err != nil {
				return fmt.Errorf("could not index last: %w", err)
			}

			log.Info().Msg("block indexed")

			m.tree = tree
			deltas = make([]dps.Delta, 0, 16)
			height++

			continue Outer

			// TODO: we should randomly run compactions during this loop as well
			// so that we still keep the DB optimized even when streaming the
			// trie updates
			// => https://github.com/awfm9/flow-dps/issues/59
		}

		// We do want to check for shutdown before pulling the next delta; both
		// because it starts a new "round" of processing, and because it could
		// enter into a tight loop until a delta becomes available.
		select {
		case <-m.stop:
			break Outer
		default:
			// keep going
		}

		delta, err := m.feed.Delta(commitTree)
		if len(deltas) == 0 && errors.Is(err, dps.ErrNotFound) {
			return fmt.Errorf("could not resolve gap, aborting")
		}
		if errors.Is(err, dps.ErrNotFound) {
			log.Warn().Msg("delta retrieval failed, rewinding")
			tree = m.tree
			deltas = make([]dps.Delta, 0, 16)
			continue Outer
		}
		if errors.Is(err, dps.ErrTimeout) {
			log.Warn().Msg("delta retrieval timed out, retrying")
			continue Outer
		}
		if errors.Is(err, dps.ErrFinished) {
			log.Debug().Msg("end of delta chain reached, stopping")
			break Outer
		}
		if err != nil {
			return fmt.Errorf("could not feed next update: %w", err)
		}

		tree, err = trie.NewTrieWithUpdatedRegisters(tree, delta.Paths(), delta.Payloads())
		if err != nil {
			return fmt.Errorf("could not update trie: %w", err)
		}

		deltas = append(deltas, delta)

		commitAfter := tree.RootHash()
		log.Info().Hex("commit_after", commitAfter).Int("num_changes", len(delta)).Msg("state trie updated")
	}

	// At the very end, we want to compact the database one more time to make
	// sure it is stored as efficiently as possible for access optimization.
	err := m.index.Compact()
	if err != nil {
		return fmt.Errorf("could not compact index: %w", err)
	}

	return nil
}
