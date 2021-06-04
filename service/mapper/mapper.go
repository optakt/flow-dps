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
	"time"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/models/index"
)

type Mapper struct {
	log        zerolog.Logger
	chain      Chain
	feed       Feeder
	index      index.Writer
	checkpoint string
	post       func(*trie.MTrie)
	wg         *sync.WaitGroup
	stop       chan struct{}
	done       chan struct{}
}

// New creates a new mapper that uses chain data to map trie updates to blocks
// and then passes on the details to the indexer for indexing.
func New(log zerolog.Logger, chain Chain, feed Feeder, index index.Writer, options ...func(*MapperConfig)) (*Mapper, error) {

	// We don't use a checkpoint by default. The options can set one, in which
	// case we will add the checkpoint as a finalized state commitment in our
	// trie registry.
	cfg := MapperConfig{
		CheckpointFile: "",
		PostProcessing: PostNoop,
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
		post:       cfg.PostProcessing,
		wg:         &sync.WaitGroup{},
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
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

func (m *Mapper) Done() <-chan struct{} {
	return m.done
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
		tree = trie.NewEmptyMTrie()
	} else {
		start := time.Now()
		m.log.Info().Time("start", start).Str("checkpoint", m.checkpoint).Msg("checkpoint rebuild starting")
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
		finish := time.Now()
		duration := finish.Sub(start)
		m.log.Info().Time("finish", finish).Str("duration", duration.Round(time.Second).String()).Msg("checkpoint rebuild finished")
	}

	// When trying to go from one finalized block to the next, we keep a list
	// of intermediary tries until the full set of transitions have been
	// identified. We keep track of these transitions as steps in this map.
	steps := make(map[flow.StateCommitment]*Step)

	// The root block does not necessarily overlap with the first state trie.
	// It's possible that we have to apply some of the trie updates before the
	// root block, in which case we map them to it. We therefore have to take
	// the initial trie's state commitment as starting point and at it to the
	// transitions as the very first one made.
	commitPrev := flow.StateCommitment(tree.RootHash())
	steps[commitPrev] = &Step{
		Commit: flow.StateCommitment{}, // not needed
		Paths:  nil,                    // not needed
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
	once := &sync.Once{}
Outer:
	for {
		// We want to check in this tight loop if we want to quit, just in case
		// we get stuck on a timed out network connection.
		select {
		case <-m.stop:
			break Outer
		default:
			// keep going
		}

		log := m.log.With().
			Uint64("height", height).
			Hex("commit_prev", commitPrev[:]).Logger()

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

		log = log.With().Hex("commit_next", commitNext[:]).Logger()

	Inner:
		for {
			// We want to check in this tight loop if we want to quit, just in case
			// we get stuck on a timed out network connection.
			select {
			case <-m.stop:
				break Outer
			default:
				// keep going
			}

			// We first look for a trie in our register whose state commitment
			// corresponds to the next block's state commitment. If we find one
			// we can break the inner loop and simply map the collected deltas
			// and the other block data to the block.
			_, ok := steps[commitNext]
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

			// We don't really need to copy this anymore, but we want to deal
			// with a single type in our code, and the value type will be copied
			// anyway.
			commitBefore := flow.StateCommitment(update.RootHash)

			log := log.With().Hex("commit_before", commitBefore[:]).Logger()

			// We now try to find the trie that this delta should be applied to.
			// If we can't find it, the delta is probably for a pruned trie and
			// we can discard it.
			step, ok := steps[commitBefore]
			if !ok {
				log.Debug().Msg("skipping trie update without matching trie")
				continue Inner
			}

			// Deduplicate the paths and payloads. We no longer need to copy the
			// path because it is now an array, and thus a value type, that is
			// copied on assignment.
			paths := make([]ledger.Path, 0, len(update.Paths))
			lookup := make(map[ledger.Path]*ledger.Payload)
			for i, path := range update.Paths {
				_, ok := lookup[path]
				if !ok {
					paths = append(paths, path)
				}
				lookup[path] = update.Payloads[i]
			}
			sort.Slice(paths, func(i, j int) bool {
				return bytes.Compare(paths[i][:], paths[j][:]) < 0
			})
			payloads := make([]ledger.Payload, 0, len(paths))
			for _, path := range paths {
				payloads = append(payloads, *lookup[path])
			}

			// Otherwise, we can apply the delta to the tree we retrieved and
			// get the resulting state commitment. We then create the step that
			// tracks our changes throughout the tries in our register
			// NOTE: It's important that we don't shadow the variable here,
			// otherwise the root trie will never go out of scope and we will
			// never garbage collect any of the initial payloads.
			tree, err = trie.NewTrieWithUpdatedRegisters(step.Tree, paths, payloads)
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
			steps[commitAfter] = step

			log.Debug().Hex("commit_after", commitAfter[:]).Msg("trie update applied")
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

		// TODO: look at performance of doing separate transactions versus
		// having an API that allows combining into a single Badger tx
		// => https://github.com/optakt/flow-dps/issues/36

		// Index all of the data for this height.
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

		// We use the tree from the step and the paths that were changed to
		// index each change for this height. As we are starting at the back, we
		// keep track of paths already updated, so we only index the last change
		// to the register payload.
		commit := commitNext
		updated := make(map[ledger.Path]struct{})
		for commit != commitPrev {
			step := steps[commit]
			paths := make([]ledger.Path, 0, len(step.Paths))
			for _, path := range step.Paths {
				_, ok := updated[path]
				if ok {
					continue
				}
				paths = append(paths, path)
				updated[path] = struct{}{}
			}
			payloads := step.Tree.UnsafeRead(paths)
			err = m.index.Payloads(height, paths, payloads)
			if err != nil {
				return fmt.Errorf("could not index payloads: %w", err)
			}
			commit = step.Commit
		}

		// At this point, we can delete any trie that does not correspond to
		// the state that we have just reached.
		for key := range steps {
			if key != commitNext {
				delete(steps, key)
			}
		}

		// The first height is only indexed once, but we always index the last
		// indexed block.
		once.Do(func() {
			err = m.index.First(height)
		})
		if err != nil {
			return fmt.Errorf("could not index first height: %w", err)
		}
		err = m.index.Last(height)
		if err != nil {
			return fmt.Errorf("could not index last height: %w", err)
		}

		// At this point, we increase the height; we have found the full
		// path of deltas to the current height and it is a finalized block,
		// so we will never look at a lower height again.
		commitPrev = commitNext
		height++

		blockID := header.ID()
		log.Info().
			Hex("block", blockID[:]).
			Int("num_changes", len(updated)).
			Int("num_events", len(events)).
			Msg("block data indexed")
	}

	step := steps[commitPrev]
	m.post(step.Tree)

	close(m.done)

	return nil
}
