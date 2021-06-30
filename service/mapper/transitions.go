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
	if !Empty(s) {
		return fmt.Errorf("invalid state for bootstrap")
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
	paths := allPaths(first)

	// We need to sort this, otherwise the retrieval from the tree later
	// when we index (with unsafe read) might fail to work properly.
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
	s.state = stateForwarded

	return nil
}

func (t *Transitions) UpdateTree(s *State) error {

	// We should only update the tree if we are ready. We are ready as long as
	// we are in the active state, but the forest does not contain the tree for
	// the state commitment of the next finalized block.
	if !Ready(s) {
		return fmt.Errorf("invalid state for update")
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

	// s.state = stateActive
	return nil
}

func (t *Transitions) IndexTree(s *State) error {

	// We should only index data if we have found the tree that corresponds
	// to the state commitment of the next finalized block.
	if !Matched(s) {
		return fmt.Errorf("invalid state for index")
	}

	// If we index payloads, we are basically stepping back from (and including)
	// the tree that corresponds to the next finalized block all the way up to
	// (and excluding) the tree for the last finalized block we indexed. To do
	// so, we will use the parent state commit to retrieve the parent trees from
	// the forest, and we use the paths we recorded changes on to retrieve the
	// changed payloads at each step.
	if t.cfg.IndexPayloads {
		commit := s.next
		updated := make(map[ledger.Path]struct{})
		for commit != s.last {

			// In the first part, we get the step we are currently at and filter
			// out any paths that have already been updated.
			ok := s.forest.Has(commit)
			if !ok {
				return fmt.Errorf("could not load tree (commit: %x)", commit)
			}

			paths, _ := s.forest.Paths(commit)
			deduplicated := make([]ledger.Path, 0, len(paths))
			for _, path := range paths {
				_, ok := updated[path]
				if ok {
					continue
				}
				deduplicated = append(deduplicated, path)
				updated[path] = struct{}{}
			}

			// We then divide the remaining paths into chunks of 1000. For each
			// batch, we retrieve the payloads from the state trie as it was at
			// the end of this block and index them.
			n := 1000
			tree, _ := s.forest.Tree(commit)
			for start := 0; start < len(deduplicated); start += n {
				end := start + n
				if end > len(deduplicated) {
					end = len(deduplicated)
				}
				batch := deduplicated[start:end]
				payloads := tree.UnsafeRead(batch)
				err := t.index.Payloads(s.height, batch, payloads)
				if err != nil {
					return fmt.Errorf("could not index payloads: %w", err)
				}
			}

			// Finally, we forward the commit to the previous trie update and
			// repeat until we have stepped all the way back to the last indexed
			// commit.
			parent, _ := s.forest.Parent(commit)
			commit = parent
		}
	}

	// Then we set the state to indexed so we get the next commit and index
	// the static data right away.
	s.state = stateIndexed

	return nil
}

func (t *Transitions) ForwardBlock(s *State) error {

	// We should only forward the height after we have just indexed the payloads
	// of a finalized block.
	if !Indexed(s) {
		return fmt.Errorf("invalid state for forwarding height")
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
	s.state = stateForwarded

	return nil
}

func (t *Transitions) IndexChain(s *State) error {

	// Indexing of chain data should only happen after we have just forwarded
	// to the next height. This is also the case after bootstrapping.
	if !Forwarded(s) {
		return fmt.Errorf("invalid state for indexing chain")
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
	s.state = stateActive

	return nil
}
