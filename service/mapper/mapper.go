// Copyright 2021 Optakt Labs OÜ
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

	"github.com/gammazero/deque"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/node"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/models/index"
)

type Mapper struct {
	log zerolog.Logger
	cfg Config

	chain Chain
	feed  Feeder
	index index.Writer

	wg   *sync.WaitGroup
	stop chan struct{}
}

// New creates a new mapper that uses chain data to map trie updates to blocks
// and then passes on the details to the indexer for indexing.
func New(log zerolog.Logger, chain Chain, feed Feeder, index index.Writer, options ...func(*Config)) (*Mapper, error) {

	// We don't use a checkpoint by default. The options can set one, in which
	// case we will add the checkpoint as a finalized state commitment in our
	// trie registry.
	cfg := Config{
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
		log:   log,
		chain: chain,
		feed:  feed,
		index: index,
		cfg:   cfg,
		wg:    &sync.WaitGroup{},
		stop:  make(chan struct{}),
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

	// We always initialize an empty state trie to refer to the first step
	// before the checkpoint. If there is no checkpoint, then the step after the
	// checkpoint will also just be the empty trie. Otherwise, the second trie
	// will load the checkpoint trie.
	empty := trie.NewEmptyMTrie()
	var tree *trie.MTrie
	if m.cfg.CheckpointFile == "" {
		tree = empty
	} else {
		m.log.Info().Msg("checkpoint rebuild started")
		file, err := os.Open(m.cfg.CheckpointFile)
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
		m.log.Info().Msg("checkpoint rebuild finished")
	}

	m.log.Info().Msg("path collection started")

	// We have to index all of the paths from the checkpoint; otherwise, we will
	// miss every single one of the bootstrapped registers.
	paths := make([]ledger.Path, 0, len(tree.AllPayloads()))
	queue := deque.New()
	root := tree.RootNode()
	if root != nil {
		queue.PushBack(root)
	}
	for queue.Len() > 0 {
		node := queue.PopBack().(*node.Node)
		if node.IsLeaf() {
			path := node.Path()
			paths = append(paths, *path)
			continue
		}
		if node.LeftChild() != nil {
			queue.PushBack(node.LeftChild())
		}
		if node.RightChild() != nil {
			queue.PushBack(node.RightChild())
		}
	}

	m.log.Info().Int("paths", len(paths)).Msg("path collection finished")

	m.log.Info().Msg("path sorting started")

	sort.Slice(paths, func(i int, j int) bool {
		return bytes.Compare(paths[i][:], paths[j][:]) < 0
	})

	m.log.Info().Msg("path sorting finished")

	// When trying to go from one finalized block to the next, we keep a list
	// of intermediary tries until the full set of transitions have been
	// identified. We keep track of these transitions as steps in this map.
	steps := make(map[flow.StateCommitment]*Step)

	// We start at an "imaginary" step that refers to an empty trie, has no
	// paths and no previous commit. We consider this step already done, so it
	// will never be indexed; it's merely used as the sentinel value for
	// stopping when we index the first block. It also makes sure that we don't
	// return a `nil` trie if we abort indexing before the first block is done.
	emptyCommit := flow.DummyStateCommitment
	steps[emptyCommit] = &Step{
		Commit: flow.StateCommitment{},
		Paths:  nil,
		Tree:   empty,
	}

	// We then add a second step that refers to the first step that is already
	// done, which uses the commit of the initial state trie after the
	// checkpoint has been loaded, and contains all of the paths found in the
	// initial checkpoint state trie. This will make sure that we index all the
	// data from the checkpoint as part of the first block.
	rootCommit := flow.StateCommitment(tree.RootHash())
	steps[rootCommit] = &Step{
		Commit: emptyCommit,
		Paths:  paths,
		Tree:   tree,
	}

	// This is how we let the indexing loop know that the first "imaginary" step
	// was already indexed. The `commitPrev` value is used as a sentinel value
	// for when to stop going backwards through the steps when indexing a block.
	// This means the value is always set to the last already indexed step.
	commitPrev := emptyCommit

	m.log.Info().Msg("state indexing started")

	// Next, we launch into the loop that is responsible for mapping all
	// incoming trie updates to a block. The loop itself has no concept of what
	// the next state commitment is that we should look at. It will simply try
	// to find a previous step for _any_ trie update that comes in. This means
	// that the first trie update needs to either apply to the empty trie or to
	// the trie after the checkpoint in order to be processed.
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

		// As a first step, we retrieve the state commitment of the finalized
		// block at the current height; we start at the root height and then
		// increase it each time we are done indexing a block. Once an applied
		// trie update gives us a state trie with the same root hash as
		// `commitNext`, we have reached the end state of the next finalized
		// block and can index all steps in-between for that block height.
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

			// When we have the state commitment of the next finalized block, we
			// check to see if we find a trie for it in our steps. If we do, it
			// means that we have steps from the last finalized block to the
			// finalized block at the current height. This condition will
			// trigger immediately for every empty block.
			_, ok := steps[commitNext]
			if ok {
				break Inner
			}

			// If we don't find a trie for the current state commitment, we need
			// to keep applying trie updates to state tries until one of them
			// does have the correct commit. We simply feed the next trie update
			// here.
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

			// NOTE: We used to require a copy of the `RootHash` here, when it
			// was still a byte slice, as the underlying slice was being reused.
			// It was changed to a value type that is always copied now.
			commitBefore := flow.StateCommitment(update.RootHash)

			log := log.With().Hex("commit_before", commitBefore[:]).Logger()

			// Once we have our new update and know which trie it should be
			// applied to, we check to see if we have such a trie in our current
			// steps. If not, we can simply skip it; this can happen, for
			// example, when there is an execution fork and the trie update
			// applies to an obsolete part of the blockchain history.
			step, ok := steps[commitBefore]
			if !ok {
				log.Debug().Msg("skipping trie update without matching trie")
				continue Inner
			}

			// We de-duplicate the paths and payloads here. This replicates some
			// code that is part of the execution node and has moved between
			// different layers of the architecture. We keep it to be safe for
			// all versions of the Flow dependencies.
			// NOTE: Past versions of this code required paths to be copied,
			// because the underlying slice was being re-used. In contrary,
			// deep-copying payloads was a bad idea, because they were already
			// being copied by the trie insertion code, and it would have led to
			// twice the memory usage.
			paths = make([]ledger.Path, 0, len(update.Paths))
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

			// We can now apply the trie update to the state trie as it was at
			// the previous step. This is where the trie code will deep-copy the
			// payloads.
			// NOTE: It's important that we don't shadow the variable here,
			// otherwise the root trie will never go out of scope and we will
			// never garbage collect any of the root trie payloads that have
			// been replaced by subsequent trie updates.
			tree, err = trie.NewTrieWithUpdatedRegisters(step.Tree, paths, payloads)
			if err != nil {
				return fmt.Errorf("could not update trie: %w", err)
			}

			// We then store the new trie along with the state commitment of its
			// parent and the paths that were changed. This will make it
			// available for subsequent trie updates to be applied to it, and it
			// will also allow us to reconstruct the payloads changed in this
			// step by retrieving them directly from the trie with the given
			// paths.
			commitAfter := flow.StateCommitment(tree.RootHash())
			step = &Step{
				Commit: commitBefore,
				Paths:  paths,
				Tree:   tree,
			}
			steps[commitAfter] = step

			log.Debug().Hex("commit_after", commitAfter[:]).Msg("trie update applied")
		}

		// At this point we have identified a step that has lead to the state
		// commitment of the finalized block at the current height. We can
		// retrieve some additional indexing data, such as the block header and
		// the events that resulted from transactions in the block.
		header, err := m.chain.Header(height)
		if err != nil {
			return fmt.Errorf("could not retrieve header: %w (height: %d)", err, height)
		}
		events, err := m.chain.Events(height)
		if err != nil {
			return fmt.Errorf("could not retrieve events: %w (height: %d)", err, height)
		}
		transactions, err := m.chain.Transactions(height)
		if err != nil {
			return fmt.Errorf("could not retrieve transactions: %w (height: %d)", err, height)
		}
		collections, err := m.chain.Collections(height)
		if err != nil {
			return fmt.Errorf("could not retrieve collections: %w (height: %d)", err, height)
		}
		blockID := header.ID()

		// TODO: Refactor the mapper in https://github.com/optakt/flow-dps/issues/128
		// and replace naive if statements around indexing.

		// We then index the data for the finalized block at the current height.
		if m.cfg.indexAll || m.cfg.indexHeaders {
			err = m.index.Header(height, header)
			if err != nil {
				return fmt.Errorf("could not index header: %w", err)
			}
		}
		if m.cfg.indexAll || m.cfg.indexRegisters {
			err = m.index.Commit(height, commitNext)
			if err != nil {
				return fmt.Errorf("could not index commit: %w", err)
			}
		}
		if m.cfg.indexAll || m.cfg.indexEvents {
			err = m.index.Events(height, events)
			if err != nil {
				return fmt.Errorf("could not index events: %w", err)
			}
		}
		if m.cfg.indexAll || m.cfg.indexBlocks {
			err = m.index.Height(blockID, height)
			if err != nil {
				return fmt.Errorf("could not index block heights: %w", err)
			}
		}
		if m.cfg.indexAll || m.cfg.indexTransactions {
			err = m.index.Transactions(blockID, collections, transactions)
			if err != nil {
				return fmt.Errorf("could not index transactions: %w", err)
			}
		}

		// In order to index the payloads, we step back from the state
		// commitment of the finalized block at the current height to the state
		// commitment of the last finalized block that was indexed. For each
		// step, we collect all the payloads by using the paths for the step and
		// index them as we go.
		// NOTE: We keep track of the paths for which we already indexed
		// payloads, so we can skip them in earlier steps. One inherent benefit
		// of stepping from the last step to the first step is that this will
		// automatically use only the latest update of a register, which is
		// exactly what we want.
		commit := commitNext
		updated := make(map[ledger.Path]struct{})
		for commit != commitPrev {

			// In the first part, we get the step we are currently at and filter
			// out any paths that have already been updated.
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

			// We then divide the remaining paths into chunks of 1000. For each
			// batch, we retrieve the payloads from the state trie as it was at
			// the end of this block and index them.
			if m.cfg.indexAll || m.cfg.indexPayloads {
				count := 0
				n := 1000
				total := (len(paths) + n - 1) / n
				log.Debug().Int("num_paths", len(paths)).Int("num_batches", total).Msg("path batching executed")
				for start := 0; start < len(paths); start += n {
					// This loop may take a while, especially for the root checkpoint
					// updates, so check if we should quit.
					select {
					case <-m.stop:
						break Outer
					default:
						// keep going
					}

					end := start + n
					if end > len(paths) {
						end = len(paths)
					}
					batch := paths[start:end]
					payloads := step.Tree.UnsafeRead(batch)
					err = m.index.Payloads(height, batch, payloads)
					if err != nil {
						return fmt.Errorf("could not index payloads: %w", err)
					}

					count++

					log.Debug().Int("batch", count).Int("start", start).Int("end", end).Msg("path batch indexed")
				}
			}

			// Finally, we forward the commit to the previous trie update and
			// repeat until we have stepped all the way back to the last indexed
			// commit.
			commit = step.Commit
		}

		// At this point, we can delete any trie that does not correspond to
		// the state that we have just reached. This will allow the garbage
		// collector to free up any payload that has been changed and which is
		// no longer part of the state trie at the newly indexed finalized
		// block.
		for key := range steps {
			if key != commitNext {
				delete(steps, key)
			}
		}

		// Last but not least, we take care of properly indexing the height of
		// the first indexed block and the height of the last indexed block.
		if m.cfg.indexAll || m.cfg.indexBlocks {
			once.Do(func() { err = m.index.First(height) })
			if err != nil {
				return fmt.Errorf("could not index first height: %w", err)
			}
			err = m.index.Last(height)
			if err != nil {
				return fmt.Errorf("could not index last height: %w", err)
			}
		}

		// We have now successfully indexed all state trie changes and other
		// data at the current height. We set the last indexed step to the last
		// step from our current height, and then increase the height to start
		// the indexing of the next block.
		commitPrev = commitNext
		height++

		log.Info().
			Hex("block", blockID[:]).
			Int("num_changes", len(updated)).
			Int("num_events", len(events)).
			Msg("block data indexed")
	}

	m.log.Info().Msg("state indexing finished")

	step := steps[commitPrev]
	m.cfg.PostProcessing(step.Tree)

	return nil
}
