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
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type TransitionFunc func(*State) error

type Transitions struct {
	cfg   Config
	log   zerolog.Logger
	chain dps.Chain
	feed  Feeder
	read  dps.Reader
	write dps.Writer
	once  *sync.Once
}

func NewTransitions(log zerolog.Logger, chain dps.Chain, feed Feeder, read dps.Reader, write dps.Writer, options ...func(*Config)) *Transitions {

	cfg := DefaultConfig
	for _, option := range options {
		option(&cfg)
	}

	t := Transitions{
		log:   log.With().Str("component", "mapper_transitions").Logger(),
		cfg:   cfg,
		chain: chain,
		feed:  feed,
		read:  read,
		write: write,
		once:  &sync.Once{},
	}

	return &t
}

func (t *Transitions) BootstrapState(s *State) error {
	if s.status != StatusBootstrap {
		return fmt.Errorf("invalid status for bootstrapping state (%s)", s.status)
	}

	// When determining which trie updates to include for a block, we always
	// track from one finalized block to the next finalized block. When we index
	// the root block, we don't have any such block. We thus add an imaginary
	// block with an empty trie that is considered the parent of the root block
	// to the forest.
	empty := trie.NewEmptyMTrie()
	s.forest.Save(empty, nil, flow.DummyStateCommitment)

	// We set it up so that it looks like the trie of our imaginary root block
	// parent was already indexed. This means that `next` is set to the commit
	// of its empty trie, while last is set to a zero-value commit.
	first := flow.StateCommitment(empty.RootHash())
	s.last = flow.DummyStateCommitment
	s.next = first

	t.log.Info().Hex("commit", first[:]).Msg("added empty tree to forest")

	// Next, we need to load the root checkpoint so that we can extract and
	// index the ledger registers paths and values at the start of the spork.
	// They will all be considered to be included in the root block.
	if t.cfg.RootCheckpoint == "" {
		return fmt.Errorf("bootstrapping requires root checkpoint file")
	}
	file, err := os.Open(t.cfg.RootCheckpoint)
	if err != nil {
		return fmt.Errorf("could not open checkpoint file: %w", err)
	}
	checkpoint, err := wal.ReadCheckpoint(file)
	if err != nil {
		return fmt.Errorf("could not read checkpoint: %w", err)
	}

	// As we said, we want to index all ledger entries from the checkpoint under
	// the root block, which means we need to start indexing at the root height.
	height, err := t.chain.Root()
	if err != nil {
		return fmt.Errorf("could not get root height: %w", err)
	}
	s.height = height

	// Rebuild the root checkpoint trie and extract all paths from it. We add
	// this to the forest, which will allow us to index all trie registers when
	// we reach the root block state commitment.
	trees, err := flattener.RebuildTries(checkpoint)
	if err != nil {
		return fmt.Errorf("could not rebuild tries: %w", err)
	}
	if len(trees) != 1 {
		return fmt.Errorf("should only have one trie in root checkpoint (tries: %d)", len(trees))
	}
	tree := trees[0]
	paths := extractPaths(tree)
	s.forest.Save(tree, paths, s.next)

	second := tree.RootHash()
	t.log.Info().Uint64("height", s.height).Hex("commit", second[:]).Int("registers", len(paths)).Msg("added checkpoint tree to forest")

	// At this point, we can check if we are operating on an empty index
	// database, or if we should skip indexing up to a certain height.
	last, err := t.read.Last()
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return fmt.Errorf("could not retrieve first: %w", err)
	}
	if err == nil {
		s.skip = last
	}

	// Now, we are ready to start indexing, respectively we skip indexing and
	// rebuild the trie up to the point where we start indexing again.
	s.status = StatusIndex

	return nil
}

func (t *Transitions) IndexChain(s *State) error {
	if s.status != StatusIndex {
		return fmt.Errorf("invalid status for indexing chain (%s)", s.status)
	}

	log := t.log.With().Uint64("height", s.height).Logger()

	// As we have only just forwarded to this height, we need to set the commit
	// of the next finalized block as the sentinel we will be looking for.
	commit, err := t.chain.Commit(s.height)
	if errors.Is(err, dps.ErrUnavailable) {
		log.Debug().Msg("waiting for next state commitment")
		time.Sleep(t.cfg.WaitInterval)
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not get commit: %w", err)
	}
	s.last = s.next
	s.next = commit

	// If we have previously indexed this height, we simply skip to the trie
	// updating at this point.
	if s.height <= s.skip {
		s.status = StatusUpdate
		return nil
	}

	// After that, we index all chain data that is configured for being indexed
	// currently.
	event := log.Info()
	err = t.write.Commit(s.height, commit)
	if err != nil {
		return fmt.Errorf("could not index commit: %w", err)
	}
	event = event.Hex("commit", commit[:])

	header, err := t.chain.Header(s.height)
	if err != nil {
		return fmt.Errorf("could not get header: %w", err)
	}
	err = t.write.Header(s.height, header)
	if err != nil {
		return fmt.Errorf("could not index header: %w", err)
	}
	blockID := header.ID()
	err = t.write.Height(blockID, s.height)
	if err != nil {
		return fmt.Errorf("could not index height: %w", err)
	}
	event = event.Hex("block", blockID[:])

	collections, err := t.chain.Collections(s.height)
	if err != nil {
		return fmt.Errorf("could not get collections: %w", err)
	}
	err = t.write.Collections(s.height, collections)
	if err != nil {
		return fmt.Errorf("could not index collections: %w", err)
	}
	event = event.Int("collections", len(collections))

	guarantees, err := t.chain.Guarantees(s.height)
	if err != nil {
		return fmt.Errorf("could not get guarantees: %w", err)
	}
	err = t.write.Guarantees(s.height, guarantees)
	if err != nil {
		return fmt.Errorf("could not index guarantees: %w", err)
	}
	event = event.Int("guarantees", len(guarantees))

	transactions, err := t.chain.Transactions(s.height)
	if err != nil {
		return fmt.Errorf("could not get transactions: %w", err)
	}
	err = t.write.Transactions(s.height, transactions)
	if err != nil {
		return fmt.Errorf("could not index transactions: %w", err)
	}
	event = event.Int("transactions", len(transactions))

	results, err := t.chain.Results(s.height)
	if err != nil {
		return fmt.Errorf("could not get transaction results: %w", err)
	}
	err = t.write.Results(results)
	if err != nil {
		return fmt.Errorf("could not index transaction results: %w", err)
	}
	event = event.Int("transaction_results", len(results))

	events, err := t.chain.Events(s.height)
	if err != nil {
		return fmt.Errorf("could not get events: %w", err)
	}
	err = t.write.Events(s.height, events)
	if err != nil {
		return fmt.Errorf("could not index events: %w", err)
	}
	event = event.Int("events", len(events))

	seals, err := t.chain.Seals(s.height)
	if err != nil {
		return fmt.Errorf("could not get seals: %w", err)
	}
	err = t.write.Seals(s.height, seals)
	if err != nil {
		return fmt.Errorf("could not index seals: %w", err)
	}
	event = event.Int("seals", len(seals))

	event.Msg("indexed blockchain data for finalized block")

	// After indexing the blockchain data, we can go back to updating the state
	// tree until we find the commit of the finalized block. This will allow us
	// to index the payloads then.
	s.status = StatusUpdate

	return nil
}

func (t *Transitions) UpdateTree(s *State) error {
	if s.status != StatusUpdate {
		return fmt.Errorf("invalid status for updating tree (%s)", s.status)
	}

	log := t.log.With().Uint64("height", s.height).Hex("last", s.last[:]).Hex("next", s.next[:]).Logger()

	// If the forest contains a tree for the commit of the next finalized block,
	// we have reached our goal and we can go to the next step in order to
	// collect the register payloads we want to index for that block.
	ok := s.forest.Has(s.next)
	if ok {
		log.Info().Hex("commit", s.next[:]).Msg("matched commit of finalized block")
		s.status = StatusCollect
		return nil
	}

	// First, we get the next tree update from the feeder. We can skip it if
	// it doesn't have any updated paths, or if we can't find the tree to apply
	// it to in the forest. This usually means that it was meant for a pruned
	// branch of the execution forest.
	update, err := t.feed.Update()
	if errors.Is(err, dps.ErrUnavailable) {
		time.Sleep(t.cfg.WaitInterval)
		log.Debug().Msg("waiting for next trie update")
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not feed update: %w", err)
	}
	parent := flow.StateCommitment(update.RootHash)
	tree, ok := s.forest.Tree(parent)
	if !ok {
		log.Warn().Msg("state commitment mismatch, retrieving next trie update")
		return nil
	}

	// We then apply the update to the relevant tree, as retrieved from the
	// forest, and save the updated tree in the forest. If the tree is not new,
	// we should error, as that should not happen.
	paths, payloads := pathsPayloads(update)
	tree, err = trie.NewTrieWithUpdatedRegisters(tree, paths, payloads)
	if err != nil {
		return fmt.Errorf("could not update tree: %w", err)
	}
	s.forest.Save(tree, paths, parent)

	hash := tree.RootHash()
	log.Info().Hex("commit", hash[:]).Int("registers", len(paths)).Msg("updated tree with register payloads")

	return nil
}

func (t *Transitions) CollectRegisters(s *State) error {
	if s.status != StatusCollect {
		return fmt.Errorf("invalid status for collecting registers (%s)", s.status)
	}

	log := t.log.With().Uint64("height", s.height).Hex("commit", s.next[:]).Logger()

	// If we already indexed this height, we simply skip to forwarding the
	// height to the next block.
	if s.height <= s.skip {
		s.status = StatusForward
		return nil
	}

	// If indexing payloads is disabled, we can bypass collection and indexing
	// of payloads and just go straight to forwarding the height to the next
	// finalized block.
	if t.cfg.SkipRegisters {
		s.status = StatusForward
		return nil
	}

	// If we index payloads, we are basically stepping back from (and including)
	// the tree that corresponds to the next finalized block all the way up to
	// (and excluding) the tree for the last finalized block we indexed. To do
	// so, we will use the parent state commit to retrieve the parent trees from
	// the forest, and we use the paths we recorded changes on to retrieve the
	// changed payloads at each step.
	commit := s.next
	for commit != s.last {

		// We do this check only once, so that we don't need to do it for
		// each item we retrieve. The tree should always be there, but we
		// should check just to not fail silently.
		ok := s.forest.Has(commit)
		if !ok {
			return fmt.Errorf("could not load tree (commit: %x)", commit)
		}

		// For each path, we retrieve the payload and add it to the registers we
		// will index later. If we already have a payload for the path, it is
		// more recent as we iterate backwards in time, so we can skip the
		// outdated payload.
		// NOTE: We read from the tree one by one here, as the performance
		// overhead is minimal compared to the disk i/o for badger, and it
		// allows us to ignore sorting of paths.
		tree, _ := s.forest.Tree(commit)
		paths, _ := s.forest.Paths(commit)
		for _, path := range paths {
			_, ok := s.registers[path]
			if ok {
				continue
			}
			payloads := tree.UnsafeRead([]ledger.Path{path})
			s.registers[path] = payloads[0]
		}

		log.Debug().Int("batch", len(paths)).Msg("collected register batch for finalized block")

		// We now step back to the parent of the current state trie.
		parent, _ := s.forest.Parent(commit)
		commit = parent
	}

	log.Info().Int("registers", len(s.registers)).Msg("collected all registers for finalized block")

	// At this point, we have collected all the payloads, so we go to the next
	// step, where we will index them.
	s.status = StatusMap

	return nil
}

func (t *Transitions) MapRegisters(s *State) error {
	if s.status != StatusMap {
		return fmt.Errorf("invalid status for indexing registers (%s)", s.status)
	}

	log := t.log.With().Uint64("height", s.height).Hex("commit", s.next[:]).Logger()

	// If there are no registers left to be indexed, we can go to the next step,
	// which is about forwarding the height to the next finalized block.
	if len(s.registers) == 0 {
		log.Info().Msg("indexed all registers for finalized block")
		s.status = StatusForward
		return nil
	}

	// We will now collect and index 1000 registers at a time. This gives the
	// FSM the chance to exit the loop between every 1000 payloads we index. It
	// doesn't really matter for badger if they are in random order, so this
	// way of iterating should be fine.
	n := 1000
	paths := make([]ledger.Path, 0, n)
	payloads := make([]*ledger.Payload, 0, n)
	for path, payload := range s.registers {
		paths = append(paths, path)
		payloads = append(payloads, payload)
		delete(s.registers, path)
		if len(paths) >= n {
			break
		}
	}

	// Then we store the (maximum) 1000 paths and payloads.
	err := t.write.Payloads(s.height, paths, payloads)
	if err != nil {
		return fmt.Errorf("could not index registers: %w", err)
	}

	log.Debug().Int("batch", len(paths)).Int("remaining", len(s.registers)).Msg("indexed register batch for finalized block")

	return nil
}

func (t *Transitions) ForwardHeight(s *State) error {
	if s.status != StatusForward {
		return fmt.Errorf("invalid status for forwarding height (%s)", s.status)
	}

	// After finishing the indexing of the payloads for a finalized block, or
	// skipping it, we should document the last indexed height. On the first
	// pass, we will also index the first indexed height here.
	var err error
	err = t.write.Last(s.height)
	if err != nil {
		return fmt.Errorf("could not index last height: %w", err)
	}
	t.once.Do(func() { err = t.write.First(s.height) })
	if err != nil {
		return fmt.Errorf("could not index first height: %w", err)
	}

	// Now that we have indexed the heights, we can forward to the next height,
	// and reset the forest to free up memory.
	s.height++
	s.forest.Reset(s.next)

	t.log.Info().Uint64("height", s.height).Msg("forwarded finalized block to next height")

	// Once the height is forwarded, we can set the status so that we index
	// the blockchain data next.
	s.status = StatusIndex

	return nil
}
