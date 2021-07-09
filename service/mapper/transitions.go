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
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type TransitionFunc func(*State) error

type Transitions struct {
	cfg   Config
	log   zerolog.Logger
	load  Loader
	chain dps.Chain
	feed  Feeder
	index dps.Writer
	once  *sync.Once
}

func NewTransitions(log zerolog.Logger, load Loader, chain dps.Chain, feed Feeder, index dps.Writer, options ...func(*Config)) *Transitions {

	cfg := DefaultConfig
	for _, option := range options {
		option(&cfg)
	}

	t := Transitions{
		log:   log,
		cfg:   cfg,
		load:  load,
		chain: chain,
		feed:  feed,
		index: index,
		once:  &sync.Once{},
	}

	return &t
}

func (t *Transitions) BootstrapState(s *State) error {

	// Bootstrapping should only happen when the state is empty.
	if s.status != StatusEmpty {
		return fmt.Errorf("invalid status for bootstrapping state (%s)", s.status)
	}

	// We always need at least one step in our forest, which is used as the
	// stopping point when indexing the payloads since the last finalized
	// block. We thus introduce an empty tree, with no paths and an
	// irrelevant previous commit.
	empty := trie.NewEmptyMTrie()
	s.forest.Save(empty, nil, flow.StateCommitment{})

	// The chain indexing will forward last to next and next to current height,
	// which will be the one for the checkpoint.
	first := flow.StateCommitment(empty.RootHash())
	s.last = flow.StateCommitment{}
	s.next = first

	t.log.Info().Hex("commit", first[:]).Msg("added empty tree to forest")

	// Then, we can load the root height and apply it to the state. That
	// will allow us to load the root blockchain data in the next step.
	height, err := t.chain.Root()
	if err != nil {
		return fmt.Errorf("could not get root height: %w", err)
	}
	s.height = height

	// Next, we will load our checkpoint tree and add it as the step
	// following the first empty tree. This will ensure that we index all
	// paths within the root tree.
	checkpoint, err := t.load.Checkpoint()
	if err != nil {
		return fmt.Errorf("could not read checkpoint: %w", err)
	}

	// Here, we store all the paths so we can index the payloads, if wanted.
	var paths []ledger.Path
	if !t.cfg.SkipBootstrap {
		paths = allPaths(checkpoint)
	}
	s.forest.Save(checkpoint, paths, first)

	second := checkpoint.RootHash()
	t.log.Info().Uint64("height", s.height).Hex("commit", second[:]).Int("registers", len(paths)).Msg("added checkpoint tree to forest")

	// We have successfully bootstrapped. However, no chain data for the root
	// block has been indexed yet. This is why we "pretend" that we just
	// forwarded the state to this height, so we go straight to the chain data
	// indexing.
	s.status = StatusForwarded

	return nil
}

func (t *Transitions) UpdateTree(s *State) error {
	log := t.log.With().Uint64("height", s.height).Hex("last", s.last[:]).Hex("next", s.next[:]).Logger()

	// We should only update the tree if we are ready. We are ready as long as
	// we are in the active state, but the forest does not contain the tree for
	// the state commitment of the next finalized block.
	if s.status != StatusUpdating {
		return fmt.Errorf("invalid status for updating tree (%s)", s.status)
	}

	// If the forest contains a tree for the commit of the next finalized block,
	// we have reached our goal and we can go to the next step in order to
	// collect the register payloads we want to index for that block.
	ok := s.forest.Has(s.next)
	if ok {
		log.Info().Hex("commit", s.next[:]).Msg("matched commit of finalized block")
		s.status = StatusMatched
		return nil
	}

	// First, we get the next tree update from the feeder. We can skip it if
	// it doesn't have any updated paths, or if we can't find the tree to apply
	// it to in the forest. This usually means that it was meant for a pruned
	// branch of the execution forest.
	update, err := t.feed.Update()
	if err != nil {
		return fmt.Errorf("could not feed update: %w", err)
	}
	parent := flow.StateCommitment(update.RootHash)
	tree, ok := s.forest.Tree(parent)
	if !ok {
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
	log := t.log.With().Uint64("height", s.height).Hex("commit", s.next[:]).Logger()

	// We should only index data if we have found the tree that corresponds
	// to the state commitment of the next finalized block.
	if s.status != StatusMatched {
		return fmt.Errorf("invalid status for collecting registers (%s)", s.status)
	}

	// If indexing payloads is disabled, we can bypass collection and indexing
	// of payloads and just go straight to forwarding the height to the next
	// finalized block.
	if !t.cfg.IndexPayloads {
		s.status = StatusIndexed
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
	s.status = StatusCollected

	return nil
}

func (t *Transitions) IndexRegisters(s *State) error {
	log := t.log.With().Uint64("height", s.height).Hex("commit", s.next[:]).Logger()

	// We should only index the payloads if we have just collected the payloads
	// of a finalized block.
	if s.status != StatusCollected {
		return fmt.Errorf("invalid status for indexing registers (%s)", s.status)
	}

	// If there are no registers left to be indexed, we can go to the next step,
	// which is about forwarding the height to the next finalized block.
	if len(s.registers) == 0 {
		log.Info().Msg("indexed all registers for finalized block")
		s.status = StatusIndexed
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
	err := t.index.Payloads(s.height, paths, payloads)
	if err != nil {
		return fmt.Errorf("could not index registers: %w", err)
	}

	log.Debug().Int("batch", len(paths)).Int("remaining", len(s.registers)).Msg("indexed register batch for finalized block")

	return nil
}

func (t *Transitions) ForwardHeight(s *State) error {

	// We should only forward the height after we have just indexed the payloads
	// of a finalized block.
	if s.status != StatusIndexed {
		return fmt.Errorf("invalid status for forwarding height (%s)", s.status)
	}

	// After finishing the indexing of the payloads for a finalized block, or
	// skipping it, we should document the last indexed height. On the first
	// pass, we will also index the first indexed height here.
	var err error
	t.once.Do(func() { err = t.index.First(s.height) })
	if err != nil {
		return fmt.Errorf("could not index first height: %w", err)
	}
	err = t.index.Last(s.height)
	if err != nil {
		return fmt.Errorf("could not index last height: %w", err)
	}

	// Now that we have indexed the heights, we can forward to the next height,
	// and reset the forest to free up memory.
	s.height++
	s.forest.Reset(s.next)

	t.log.Info().Uint64("height", s.height).Msg("forwarded finalized block to next height")

	// Once the height is forwarded, we can set the status so that we index
	// the blockchain data next.
	s.status = StatusForwarded

	return nil
}

func (t *Transitions) IndexChain(s *State) error {
	log := t.log.Info().Uint64("height", s.height)

	// Indexing of chain data should only happen after we have just forwarded
	// to the next height. This is also the case after bootstrapping.
	if s.status != StatusForwarded {
		return fmt.Errorf("invalid status for indexing chain (%s)", s.status)
	}

	// As we have only just forwarded to this height, we need to set the commit
	// of the next finalized block as the sentinal we will be looking for.
	commit, err := t.chain.Commit(s.height)
	if err != nil {
		return fmt.Errorf("could not get commit: %w", err)
	}
	s.last = s.next
	s.next = commit

	// After that, we index all chain data that is configured for being indexed
	// currently.
	if t.cfg.IndexCommit {
		err := t.index.Commit(s.height, commit)
		if err != nil {
			return fmt.Errorf("could not index commit: %w", err)
		}
		log = log.Hex("commit", commit[:])
	}
	if t.cfg.IndexHeader {
		header, err := t.chain.Header(s.height)
		if err != nil {
			return fmt.Errorf("could not get header: %w", err)
		}
		err = t.index.Header(s.height, header)
		if err != nil {
			return fmt.Errorf("could not index header: %w", err)
		}
		blockID := header.ID()
		log = log.Hex("block", blockID[:])
	}
	if t.cfg.IndexCollections {
		collections, err := t.chain.Collections(s.height)
		if err != nil {
			return fmt.Errorf("could not get collections: %w", err)
		}
		err = t.index.Collections(s.height, collections)
		if err != nil {
			return fmt.Errorf("could not index collections: %w", err)
		}
		log = log.Int("collections", len(collections))
	}
	if t.cfg.IndexTransactions {
		transactions, err := t.chain.Transactions(s.height)
		if err != nil {
			return fmt.Errorf("could not get transactions: %w", err)
		}
		err = t.index.Transactions(s.height, transactions)
		if err != nil {
			return fmt.Errorf("could not index transactions: %w", err)
		}
		log = log.Int("transactions", len(transactions))
	}
	if t.cfg.IndexEvents {
		events, err := t.chain.Events(s.height)
		if err != nil {
			return fmt.Errorf("could not get events: %w", err)
		}
		err = t.index.Events(s.height, events)
		if err != nil {
			return fmt.Errorf("could not index events: %w", err)
		}
		log = log.Int("events", len(events))
	}

	log.Msg("indexed blockchain data for finalized block")

	// After indexing the blockchain data, we can go back to updating the state
	// tree until we find the commit of the finalized block. This will allow us
	// to index the payloads then.
	s.status = StatusUpdating

	return nil
}
