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
	"fmt"
	"sort"
	"sync"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/index"
)

type TransitionFunc func(*State) error

type Transitions struct {
	cfg   Config
	load  Loader
	chain Chain
	feed  Feeder
	index index.Writer
	once  *sync.Once
}

func NewTransitions(load Loader, chain Chain, feed Feeder, index index.Writer, options ...func(*Config)) *Transitions {

	cfg := DefaultConfig
	for _, option := range options {
		option(&cfg)
	}

	t := Transitions{
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
	ok := s.forest.Save(empty, nil, flow.StateCommitment{})
	if !ok {
		return fmt.Errorf("could not save empty tree")
	}
	parent := flow.StateCommitment(empty.RootHash())
	s.last = parent

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
	first, err := t.load.Checkpoint()
	if err != nil {
		return fmt.Errorf("could not read checkpoint: %w", err)
	}

	// We need to sort this, otherwise the retrieval from the tree later
	// when we index (with unsafe read) might fail to work properly.
	paths := allPaths(first)
	sort.Slice(paths, func(i int, j int) bool {
		return bytes.Compare(paths[i][:], paths[j][:]) < 0
	})

	// Now that we have the path, we are ready to add the checkpoint step.
	ok = s.forest.Save(first, paths, parent)
	if !ok {
		return fmt.Errorf("could not save checkpoint tree")
	}

	// We have now successfully bootstrapped. However, no blockchain data
	// has been indexed, so we pretend like we just forwarded from the last
	// indexed block.
	s.status = StatusForwarded

	return nil
}

func (t *Transitions) UpdateTree(s *State) error {

	// We should only update the tree if we are ready. We are ready as long as
	// we are in the active state, but the forest does not contain the tree for
	// the state commitment of the next finalized block.
	if s.status != StatusUpdating {
		return fmt.Errorf("invalid status for updating tree (%s)", s.status)
	}

	// If we have matched the tree with the next commit, we can go to the next
	// state immediately.
	ok := s.forest.Has(s.next)
	if ok {
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
	if len(update.Paths) == 0 {
		return fmt.Errorf("empty trie update")
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
	ok = s.forest.Save(tree, paths, parent)
	if !ok {
		return fmt.Errorf("duplicate tree save")
	}

	return nil
}

func (t *Transitions) CollectRegisters(s *State) error {

	// We should only index data if we have found the tree that corresponds
	// to the state commitment of the next finalized block.
	if s.status != StatusMatched {
		return fmt.Errorf("invalid status for collecting registers (%s)", s.status)
	}

	// If we index payloads, we are basically stepping back from (and including)
	// the tree that corresponds to the next finalized block all the way up to
	// (and excluding) the tree for the last finalized block we indexed. To do
	// so, we will use the parent state commit to retrieve the parent trees from
	// the forest, and we use the paths we recorded changes on to retrieve the
	// changed payloads at each step.
	if t.cfg.IndexPayloads {
		commit := s.next
		for commit != s.last {

			// We do this check only once, so that we don't need to do it for
			// each item we retrieve. The tree should always be there, but we
			// should check just to not fail silently.
			ok := s.forest.Has(commit)
			if !ok {
				return fmt.Errorf("could not load tree (commit: %x)", commit)
			}

			// For each path, we retrieve the payload and add it to the
			// registers we want to save later. If we already have a payload,
			// we only keep that one, because we start with the newest one and
			// previous ones within the same block are thus irrelevant.
			paths, _ := s.forest.Paths(commit)
			tree, _ := s.forest.Tree(commit)
			for _, path := range paths {
				_, ok := s.registers[path]
				if ok {
					continue
				}
				payloads := tree.UnsafeRead([]ledger.Path{path})
				s.registers[path] = payloads[0]
			}

			// We now step back to the parent of the current state trie.
			parent, _ := s.forest.Parent(commit)
			commit = parent
		}
	}

	// Then we set the state to indexed so we get the next commit and index
	// the static data right away.
	s.status = StatusCollected

	return nil
}

func (t *Transitions) IndexRegisters(s *State) error {

	// We should only index the payloads if we have just collected the payloads
	// of a finalized block.
	if s.status != StatusCollected {
		return fmt.Errorf("invalid status for indexing registers (%s)", s.status)
	}

	// If there are no registers to be indexed, we can go to the next state
	// immediately.
	if len(s.registers) == 0 {
		s.status = StatusIndexed
		return nil
	}

	// We will now collect and index 1000 registers at a time. This gives the
	// FSM the chance to exit the loop between every 1000 payloads we index.
	paths := make([]ledger.Path, 0, 1000)
	payloads := make([]*ledger.Payload, 0, 1000)
	for path, payload := range s.registers {
		paths = append(paths, path)
		payloads = append(payloads, payload)
		delete(s.registers, path)
	}

	// Then we store the (maximum) 1000 paths and payloads.
	err := t.index.Payloads(s.height, paths, payloads)
	if err != nil {
		return fmt.Errorf("could not index registers: %w", err)
	}

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

	// After forwarding the height, we are in the forwarded state, which will
	// always lead into the indexing of chain data next.
	s.status = StatusForwarded

	return nil
}

func (t *Transitions) IndexChain(s *State) error {

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
		err = t.index.Commit(s.height, commit)
		if err != nil {
			return fmt.Errorf("could not index commit: %w", err)
		}
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
	}

	// At this point, all stable data for this block height is indexed and we
	// can go into the active state to start collecting payloads for the height
	// from tree updates.
	s.status = StatusUpdating

	return nil
}
