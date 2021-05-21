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
	"sort"
	"sync"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
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
	checkpoint string
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

	// Check if the checkpoint file exists.
	if cfg.CheckpointFile != "" {
		stat, err := os.Stat(cfg.CheckpointFile)
		if err != nil {
			return nil, fmt.Errorf("invalid checkpoint file: %w", err)
		}
		if stat.IsDir() {
			return nil, fmt.Errorf("invalid checkpoint file: directory")
		}
	}

	i := Mapper{
		log:        log,
		chain:      chain,
		feed:       feed,
		index:      index,
		checkpoint: cfg.CheckpointFile,
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
	height, err := m.chain.Root()
	if err != nil {
		return fmt.Errorf("could not get root height: %w", err)
	}

	// If we have no checkpoint file, we start from an empty trie; otherwise we
	// rebuild the checkpoint and use that as the starting trie.
	var tree *trie.MTrie
	if m.checkpoint == "" {
		tree, err = trie.NewEmptyMTrie(pathfinder.PathByteSize)
		if err != nil {
			return fmt.Errorf("could not initialize empty trie: %w", err)
		}
	} else {
		file, err := os.Open(m.checkpoint)
		if err != nil {
			return fmt.Errorf("could not open checkpoint file: %w", err)
		}
		checkpoint, err := wal.ReadCheckpoint(file)
		if err != nil {
			return fmt.Errorf("could not read checkpoint: %w", err)
		}
		trees, err := flattener.RebuildTries(checkpoint)
		if err != nil {
			return fmt.Errorf("could not rebuild tries: %w", err)
		}
		if len(trees) != 1 {
			return fmt.Errorf("should only have one trie in root checkpoint (tries: %d)", len(trees))
		}
		tree = trees[0]
	}

	// When trying to go from one finalized block to the next, we keep a list
	// of intermediary tries until the full set of transitions have been
	// identified. We keep track of these transitions as steps in this map.
	steps := make(map[string]*Step)

	// The root block does not necessarily overlap with the first state trie.
	// It's possible that we have to apply some of the trie updates before the
	// root block, in which case we map them to it. We therefore have to take
	// the initial trie's state commitment as starting point and at it to the
	// transitions as the very first one made.
	commitLast := flow.StateCommitment(tree.RootHash())
	steps[string(commitLast)] = &Step{
		Commit: nil, // not needed
		Paths:  nil, // not needed
		Tree:   tree,
	}

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
			Hex("commit_last", commitLast).Logger()

		// As a first step, we retrieve the state commitment of the current
		// block, which will serve as the sentinel value we will look for from
		// the state trie.
		commitNext, err := m.chain.Commit(height)

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

		log = log.With().Hex("commit_next", commitNext).Logger()

	Inner:
		for {
			// We first look for a trie in our register whose state commitment
			// corresponds to the next block's state commitment. If we find one
			// we can break the inner loop and simply map the collected deltas
			// and the other block data to the block.
			_, ok := steps[string(commitNext)]
			if ok {
				break Inner
			}

			// If we don't find a trie for the current sentinel state commitment
			// in our register of tries, we keep applying deltas.
			update, err := m.feed.Update()

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

			// We need to copy the root hash because it's still part of the
			// WAL record that will be overwritten on the next read.
			commitBefore := make(flow.StateCommitment, len(update.RootHash))
			copy(commitBefore, update.RootHash)

			log := log.With().Hex("commit_before", commitBefore).Logger()

			// We now try to find the trie that this delta should be applied to.
			// If we can't find it, the delta is probably for a pruned trie and
			// we can discard it.
			step, ok := steps[string(commitBefore)]
			if !ok {
				log.Debug().Msg("skipping trie update without matching trie")
				continue Inner
			}

			// Deduplicate the paths and payloads. We need to deep copy the path
			// because we want to keep it to retrieve the payloads later, and it
			// is still part of the WAL record that will be overwritten on the
			// next read. We don't need to deep copy the values, as the tree
			// internally already does this.
			paths := make([]ledger.Path, 0, len(update.Paths))
			lookup := make(map[string]*ledger.Payload)
			for i, path := range update.Paths {
				_, ok := lookup[string(path)]
				if !ok {
					paths = append(paths, path.DeepCopy())
				}
				lookup[string(path)] = update.Payloads[i]
			}
			sort.Slice(paths, func(i, j int) bool {
				return bytes.Compare(paths[i], paths[j]) < 0
			})
			payloads := make([]ledger.Payload, 0, len(paths))
			for _, path := range paths {
				payloads = append(payloads, *lookup[string(path)])
			}

			// Otherwise, we can apply the delta to the tree we retrieved and
			// get the resulting state commitment. We then create the step that
			// tracks our changes throughout the tries in our register
			tree, err := trie.NewTrieWithUpdatedRegisters(step.Tree, paths, payloads)
			if err != nil {
				return fmt.Errorf("could not update trie: %w", err)
			}

			// We then store the new tree along with the state commitment of its
			// parent and the paths that were changed so we can rebuild the
			// delta later.
			commitAfter := flow.StateCommitment(tree.RootHash())
			step = &Step{
				Commit: commitBefore,
				Paths:  paths,
				Tree:   tree,
			}
			steps[string(commitAfter)] = step

			log.Info().Hex("commit_after", commitAfter).Msg("trie update applied")
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

		// We use the tree from the step and the paths that were changed to
		// index each change for this height. As we are starting at the back, we
		// keep track of paths already updated, so we only index the last change
		// to the register payload.
		commit := commitNext
		updated := make(map[string]struct{})
		for !bytes.Equal(commit, commitLast) {
			step := steps[string(commit)]
			payloads := step.Tree.UnsafeRead(step.Paths)
			for i, path := range step.Paths {
				_, ok := updated[string(path)]
				if ok {
					continue
				}
				err = m.index.Payload(height, path, payloads[i])
				if err != nil {
					return fmt.Errorf("could not index payload (height: %d, path: %x): %w", height, path, err)
				}
				updated[string(path)] = struct{}{}
			}
			commit = step.Commit
		}

		// At this point, we can delete any trie that does not correspond to
		// the state that we have just reached.
		for key := range steps {
			if key != string(commitNext) {
				delete(steps, key)
			}
		}

		// TODO: look at performance of doing separate transactions versus
		// having an API that allows combining into a single Badger tx
		// => https://github.com/awfm9/flow-dps/issues/36

		// If we successfully indexed all of the deltas, we can index the rest
		// of the block data.
		err = m.index.Header(height, header)
		if err != nil {
			return fmt.Errorf("could not index header: %w", err)
		}
		err = m.index.Commit(height, commitNext)
		if err != nil {
			return fmt.Errorf("could not index commit: %w", err)
		}
		err = m.index.Events(height, events)
		if err != nil {
			return fmt.Errorf("could not index events: %w", err)
		}
		err = m.index.Last(commitNext)
		if err != nil {
			return fmt.Errorf("could not index last: %w", err)
		}

		// At this point, we increase the height; we have found the full
		// path of deltas to the current height and it is a finalized block,
		// so we will never look at a lower height again.
		commitLast = commitNext
		height++

		blockID := header.ID()
		log.Info().
			Hex("block", blockID[:]).
			Int("num_changes", len(updated)).
			Int("num_events", len(events)).
			Msg("block data indexed")
	}

	return nil
}
