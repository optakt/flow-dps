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
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
)

type Mapper struct {
	log        zerolog.Logger
	chain      Chain
	feed       Feeder
	index      dps.Index
	rootHeight uint64
	rootTree   *trie.MTrie
	wg         *sync.WaitGroup
	stop       chan struct{}
}

// New creates a new mapper that uses chain data to map trie updates to blocks
// and then passes on the details to the indexer for indexing.
func New(log zerolog.Logger, chain Chain, feed Feeder, index dps.Index, options ...func(*MapperConfig)) (*Mapper, error) {

	// We don't use a checkpoint by default. The options can set one, in which
	// case we will add the checkpoint as a finalized state commitment in our
	// trie registry.
	cfg := MapperConfig{
		CheckpointFile: "",
	}
	for _, option := range options {
		option(&cfg)
	}

	// Check that we can get a root height from chain for initialization.
	rootHeight, err := chain.Root()
	if err != nil {
		return nil, fmt.Errorf("could not get root height: %w", err)
	}

	// We create an empty trie as a default to start from. If a checkpoint file
	// is provided, we replace it with the tree rebuilt from the checkpoint
	// instead.
	rootTree, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
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
		trees, err := flattener.RebuildTries(checkpoint)
		if err != nil {
			return nil, fmt.Errorf("could not rebuild tries: %w", err)
		}
		if len(trees) != 1 {
			return nil, fmt.Errorf("should only have one trie in root checkpoint (tries: %d)", len(trees))
		}
		rootTree = trees[0]
	}

	i := Mapper{
		log:        log,
		chain:      chain,
		feed:       feed,
		index:      index,
		rootHeight: rootHeight,
		rootTree:   rootTree,
		wg:         &sync.WaitGroup{},
		stop:       make(chan struct{}),
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

// NOTE: We might want to move height and tree (checkpoint) to parameters of the
// run function; that would make it quite easy to resume from an arbitrary
// point in the LedgerWAL and get rid of the related struct fields.

func (m *Mapper) Run() error {
	m.wg.Add(1)
	defer m.wg.Done()

	// We start trying to map at the root height.
	height := m.rootHeight

	// When trying to go from one finalized block to the next, we keep a list
	// of intermediary tries until the connection has been made. We also need
	// to point to the previous trie and keep the changes that were required to
	// reach it.
	steps := make(map[string]*Step)

	// The root tree is the starting point for our mapping. It will either
	// correspond to the state at the root block, or to a state before it, in
	// which case we already need to map some deltas to the root block.
	lastCommit := flow.StateCommitment(m.rootTree.RootHash())
	steps[string(lastCommit)] = &Step{
		Delta: &dps.Delta{}, // not needed
		Tree:  m.rootTree,
	}
	m.rootTree = nil

	// The purpose of this function is to map state deltas from a continuous
	// feed to specific blocks from the chain. This is necessary because the
	// trie updates that we receive as state deltas are agnostic of blocks and
	// instead operate on a chunk level. We keep applying these deltas to our
	// set of tries until we found a path from one finalized block's sealed
	// state commitment to the state commitment of the next finalized block. At
	// that point, we can discard all other tries, as the path must continue
	// from the state trie with the sealed state commitment.
Outer:
	for {

		log := m.log.With().
			Uint64("height", height).
			Hex("last_commit", lastCommit).Logger()

		// As a first step, we retrieve the state commitment of the current
		// block, which will serve as the sentinel value we will look for from
		// the state trie.
		nextCommit, err := m.chain.Commit(height)

		// If the retrieval times out, it's possible that we are on a live chain
		// and the next block has not been finalized yet. We should thus simply
		// retry until we have a new block.
		if errors.Is(err, dps.ErrTimeout) {
			log.Warn().Msg("commit retrieval timed out, retrying")
			continue Outer
		}

		// If we have reached the end of the finalized blocks, we are probably
		// on a historical chain and there are no more finalized blocks for the
		// related spork. We can exit without error.
		if errors.Is(err, dps.ErrFinished) {
			log.Debug().Msg("reached end of finalized chain")
			break Outer
		}

		// Any other error should not happen and should crash explicitly.
		if err != nil {
			return fmt.Errorf("could not retrieve next commit (height: %d): %w", height, err)
		}

		log = log.With().Hex("next_commit", nextCommit).Logger()

	Inner:
		for {
			// We first look for a tree in our register whose state commitment
			// corresponds for our current sentinel state commitment. If we
			// find one, we break this loop and simply map the collected deltas
			// and the remaining block data to the block. This also addresses
			// the edge case of blocks without changes, which will always bypass
			// this loop on the first iteration.
			_, ok := steps[string(nextCommit)]
			if ok {
				break Inner
			}

			// If we don't find a trie for the current sentinel state commitment
			// in our register of tries, we keep applying deltas.
			delta, err := m.feed.Delta()

			// Once more, we might be on a live spork and the next delta might not
			// be available yet. In that case, keep trying.
			if errors.Is(err, dps.ErrTimeout) {
				log.Warn().Msg("delta retrieval timed out, retrying")
				continue Inner
			}

			// Similarly, if no more deltas are available, we reached the end of
			// the WAL and we are done reconstructing the execution state.
			if errors.Is(err, dps.ErrFinished) {
				log.Debug().Msg("reached end of delta log")
				break Outer
			}

			// Other errors should fail execution as they should not happen.
			if err != nil {
				return fmt.Errorf("could not retrieve next delta: %w", err)
			}

			// We now try to find the trie that this delta should be applied to.
			// If we can't find it, the delta is probably for a pruned trie and
			// we can discard it.
			step, ok := steps[string(delta.Commit)]
			if !ok {
				continue Inner
			}

			// Otherwise, we can apply the delta to the tree we retrieved and
			// get the resulting state commitment. We then create the step that
			// tracks our changes throughout the tries in our register.
			tree, err := trie.NewTrieWithUpdatedRegisters(step.Tree, delta.Paths(), delta.Payloads())
			if err != nil {
				return fmt.Errorf("could not update trie: %w", err)
			}

			treeCommit := flow.StateCommitment(tree.RootHash())
			step = &Step{
				Delta: delta,
				Tree:  tree,
			}
			steps[string(treeCommit)] = step

			log.Debug().Hex("tree_commit", treeCommit).Msg("state delta applied")
		}

		// At this point we have identified a step that has lead to the next
		// finalized state commitment. We can thus retrieve the remaining data
		// we need to fully index the block first.
		header, err := m.chain.Header(height)
		if err != nil {
			return fmt.Errorf("could not retrieve header: %w (height: %d)", err, height)
		}
		events, err := m.chain.Events(height)
		if err != nil {
			return fmt.Errorf("could not retrieve events: %w (height: %d)", err, height)
		}

		// We collect all of the deltas from the next commit that we have now
		// found, back to the last finalized commit. This will break right away
		// if the last and next commits are the same, which works well for
		// blocks that didn't change the state.
		var deltas []*dps.Delta
		commit := nextCommit
		for !bytes.Equal(commit, lastCommit) {
			step := steps[string(commit)]
			deltas = append(deltas, step.Delta)
			commit = step.Delta.Commit
		}

		// TODO: look at performance of doing separate transactions versus
		// having an API that allows combining into a single Badger tx
		// => https://github.com/awfm9/flow-dps/issues/36

		// At this point, we can delete any trie that does not correspond to
		// the state that we have just reached.
		for key := range steps {
			if key == string(nextCommit) {
				continue
			}
			delete(steps, key)
		}

		// If we successfully collected the deltas and data, we can proceed to
		// indexing it in our database.
		err = m.index.Header(height, header)
		if err != nil {
			return fmt.Errorf("could not index header: %w", err)
		}
		err = m.index.Commit(height, nextCommit)
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
		err = m.index.Last(nextCommit)
		if err != nil {
			return fmt.Errorf("could not index last: %w", err)
		}

		// At this point, we increase the height; we have found the full
		// path of deltas to the current height and it is a finalized block,
		// so we will never look at a lower height again.
		lastCommit = nextCommit
		height++

		blockID := header.ID()
		log.Info().
			Hex("block", blockID[:]).
			Int("num_deltas", len(deltas)).
			Int("num_events", len(events)).
			Msg("block data indexed")
	}

	return nil
}
