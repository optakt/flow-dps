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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/models/dps"

	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewTransitions(t *testing.T) {
	t.Run("nominal case, without options", func(t *testing.T) {
		root := trie.NewEmptyMTrie()
		chain := mocks.BaselineChain(t)
		feed := mocks.BaselineFeeder(t)
		index := mocks.BaselineWriter(t)

		tr := NewTransitions(mocks.NoopLogger, root, chain, feed, index)

		assert.NotNil(t, tr)
		assert.Equal(t, root, tr.root)
		assert.Equal(t, chain, tr.chain)
		assert.Equal(t, feed, tr.feed)
		assert.Equal(t, index, tr.index)
		assert.NotNil(t, tr.once)
		assert.Equal(t, DefaultConfig, tr.cfg)
	})

	t.Run("nominal case, with options", func(t *testing.T) {
		root := trie.NewEmptyMTrie()
		chain := mocks.BaselineChain(t)
		feed := mocks.BaselineFeeder(t)
		index := mocks.BaselineWriter(t)

		tr := NewTransitions(mocks.NoopLogger, root, chain, feed, index,
			WithIndexCommit(true),
			WithIndexHeader(true),
			WithIndexPayloads(true),
			WithIndexCollections(true),
			WithIndexGuarantees(true),
			WithIndexTransactions(true),
			WithIndexResults(true),
			WithIndexSeals(true),
		)

		assert.NotNil(t, tr)
		assert.Equal(t, root, tr.root)
		assert.Equal(t, chain, tr.chain)
		assert.Equal(t, feed, tr.feed)
		assert.Equal(t, index, tr.index)
		assert.NotNil(t, tr.once)

		assert.NotEqual(t, DefaultConfig, tr.cfg)
		assert.Equal(t, DefaultConfig.IndexEvents, tr.cfg.IndexEvents)
		assert.True(t, tr.cfg.IndexTransactions)
		assert.True(t, tr.cfg.IndexHeader)
		assert.True(t, tr.cfg.IndexPayloads)
		assert.True(t, tr.cfg.IndexCollections)
		assert.True(t, tr.cfg.IndexGuarantees)
		assert.True(t, tr.cfg.IndexTransactions)
		assert.True(t, tr.cfg.IndexResults)
		assert.True(t, tr.cfg.IndexSeals)
	})
}

func TestTransitions_BootstrapState(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusEmpty)

		// Copy state in local scope so that we can override its SaveFunc without impacting other
		// tests running in parallel.
		var saveCalled bool
		forest := mocks.BaselineForest(t, true)
		forest.SaveFunc = func(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment) {
			if !saveCalled {
				assert.True(t, tree.IsEmpty())
				assert.Nil(t, paths)
				assert.Zero(t, parent)
				saveCalled = true
				return
			}
			assert.False(t, tree.IsEmpty())
			assert.Len(t, tree.AllPayloads(), len(paths))
			assert.Len(t, paths, 3) // Expect the three paths from leaves.
			assert.NotZero(t, parent)
		}

		err := tr.BootstrapState(st)
		assert.NoError(t, err)
	})

	t.Run("invalid state", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusIndexed)

		err := tr.BootstrapState(st)
		assert.Error(t, err)
	})

	t.Run("handles failure to get root height", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusEmpty)

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		tr.chain = chain

		err := tr.BootstrapState(st)
		assert.Error(t, err)
	})
}

func TestTransitions_UpdateTree(t *testing.T) {
	update := mocks.GenericTrieUpdate(0)
	tree := mocks.GenericTrie

	t.Run("nominal case without match", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdating)

		forest := mocks.BaselineForest(t, false)
		forest.SaveFunc = func(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment) {
			// Parent is RootHash of the mocks.GenericTrie.
			assert.Equal(t, update.RootHash[:], parent[:])
			assert.ElementsMatch(t, paths, update.Paths)
			assert.NotZero(t, tree)
		}
		forest.TreeFunc = func(commit flow.StateCommitment) (*trie.MTrie, bool) {
			assert.Equal(t, update.RootHash[:], commit[:])

			return tree, true
		}
		st.forest = forest

		err := tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, StatusUpdating, st.status)
	})

	t.Run("nominal case with no available update temporarily", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdating)

		// Set up the mock feeder to return an unavailable error on the first call and return successfully
		// to subsequent calls.
		var updateCalled bool
		feeder := mocks.BaselineFeeder(t)
		feeder.UpdateFunc = func() (*ledger.TrieUpdate, error) {
			if !updateCalled {
				updateCalled = true
				return nil, dps.ErrUnavailable
			}
			return update, nil
		}
		tr.feed = feeder

		forest := mocks.BaselineForest(t, true)
		forest.HasFunc = func(flow.StateCommitment) bool {
			return updateCalled
		}
		st.forest = forest

		// The first call should not error but should not change the status of the FSM to updating. It should
		// instead remain Updating until a match is found.
		err := tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, StatusUpdating, st.status)

		// The second call is now successful and matches.
		err = tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, StatusMatched, st.status)
	})

	t.Run("nominal case with match", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdating)

		err := tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, StatusMatched, st.status)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusEmpty)

		err := tr.UpdateTree(st)

		assert.Error(t, err)
	})

	t.Run("handles feeder update failure", func(t *testing.T) {
		t.Parallel()

		feed := mocks.BaselineFeeder(t)
		feed.UpdateFunc = func() (*ledger.TrieUpdate, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusUpdating)
		st.forest = mocks.BaselineForest(t, false)
		tr.feed = feed

		err := tr.UpdateTree(st)

		assert.Error(t, err)
	})

	t.Run("handles forest parent tree not found", func(t *testing.T) {
		t.Parallel()

		forest := mocks.BaselineForest(t, false)
		forest.TreeFunc = func(_ flow.StateCommitment) (*trie.MTrie, bool) {
			return nil, false
		}

		tr, st := baselineFSM(t, StatusUpdating)
		st.forest = forest

		err := tr.UpdateTree(st)

		assert.NoError(t, err)
	})
}

func TestTransitions_CollectRegisters(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		forest := mocks.BaselineForest(t, true)
		forest.ParentFunc = func(commit flow.StateCommitment) (flow.StateCommitment, bool) {
			assert.Equal(t, mocks.GenericCommit(0), commit)

			return mocks.GenericCommit(1), true
		}

		tr, st := baselineFSM(t, StatusMatched)
		st.forest = forest

		err := tr.CollectRegisters(st)

		require.NoError(t, err)
		assert.Equal(t, StatusCollected, st.status)
		for _, wantPath := range mocks.GenericLedgerPaths(6) {
			assert.Contains(t, st.registers, wantPath)
		}
	})

	t.Run("indexing payloads disabled", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusMatched)
		tr.cfg.IndexPayloads = false

		err := tr.CollectRegisters(st)

		require.NoError(t, err)
		assert.Empty(t, st.registers)
		assert.Equal(t, StatusIndexed, st.status)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusEmpty)

		err := tr.CollectRegisters(st)

		assert.Error(t, err)
		assert.Empty(t, st.registers)
	})

	t.Run("handles missing tree for commit", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusMatched)
		st.forest = mocks.BaselineForest(t, false)

		err := tr.CollectRegisters(st)

		assert.Error(t, err)
		assert.Empty(t, st.registers)
	})
}

func TestTransitions_IndexRegisters(t *testing.T) {
	t.Run("nominal case with registers to index", func(t *testing.T) {
		t.Parallel()

		// Path 2 and 4 are the same so the map effectively contains 5 entries.
		testRegisters := map[ledger.Path]*ledger.Payload{
			mocks.GenericLedgerPath(0): mocks.GenericLedgerPayload(0),
			mocks.GenericLedgerPath(1): mocks.GenericLedgerPayload(1),
			mocks.GenericLedgerPath(2): mocks.GenericLedgerPayload(2),
			mocks.GenericLedgerPath(3): mocks.GenericLedgerPayload(3),
			mocks.GenericLedgerPath(4): mocks.GenericLedgerPayload(4),
			mocks.GenericLedgerPath(5): mocks.GenericLedgerPayload(5),
		}

		index := mocks.BaselineWriter(t)
		index.PayloadsFunc = func(height uint64, paths []ledger.Path, value []*ledger.Payload) error {
			assert.Equal(t, mocks.GenericHeight, height)

			// Expect the 5 entries from the map.
			assert.Len(t, paths, 6)
			assert.Len(t, value, 6)
			return nil
		}

		tr, st := baselineFSM(t, StatusCollected)
		tr.index = index
		st.registers = testRegisters

		err := tr.IndexRegisters(st)

		require.NoError(t, err)

		// Should not be StateIndexed because registers map was not empty.
		assert.Empty(t, st.registers)
		assert.Equal(t, StatusCollected, st.status)
	})

	t.Run("nominal case no more registers left to index", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusCollected)

		err := tr.IndexRegisters(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusIndexed, st.status)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		testRegisters := map[ledger.Path]*ledger.Payload{
			mocks.GenericLedgerPath(0): mocks.GenericLedgerPayload(0),
			mocks.GenericLedgerPath(1): mocks.GenericLedgerPayload(1),
			mocks.GenericLedgerPath(2): mocks.GenericLedgerPayload(2),
			mocks.GenericLedgerPath(3): mocks.GenericLedgerPayload(3),
			mocks.GenericLedgerPath(4): mocks.GenericLedgerPayload(4),
			mocks.GenericLedgerPath(5): mocks.GenericLedgerPayload(5),
		}

		tr, st := baselineFSM(t, StatusEmpty)
		st.registers = testRegisters

		err := tr.IndexRegisters(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure", func(t *testing.T) {
		t.Parallel()

		testRegisters := map[ledger.Path]*ledger.Payload{
			mocks.GenericLedgerPath(0): mocks.GenericLedgerPayload(0),
			mocks.GenericLedgerPath(1): mocks.GenericLedgerPayload(1),
			mocks.GenericLedgerPath(2): mocks.GenericLedgerPayload(2),
			mocks.GenericLedgerPath(3): mocks.GenericLedgerPayload(3),
			mocks.GenericLedgerPath(4): mocks.GenericLedgerPayload(4),
			mocks.GenericLedgerPath(5): mocks.GenericLedgerPayload(5),
		}

		index := mocks.BaselineWriter(t)
		index.PayloadsFunc = func(uint64, []ledger.Path, []*ledger.Payload) error { return mocks.GenericError }

		tr, st := baselineFSM(t, StatusCollected)
		tr.index = index
		st.registers = testRegisters

		err := tr.IndexRegisters(st)

		assert.Error(t, err)
	})
}

func TestTransitions_ForwardHeight(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var (
			firstCalled int
			lastCalled  int
		)
		index := mocks.BaselineWriter(t)
		index.FirstFunc = func(height uint64) error {
			assert.Equal(t, mocks.GenericHeight, height)
			firstCalled++
			return nil
		}
		index.LastFunc = func(height uint64) error {
			assert.Equal(t, mocks.GenericHeight+uint64(lastCalled), height)
			lastCalled++
			return nil
		}

		forest := mocks.BaselineForest(t, true)
		forest.ResetFunc = func(finalized flow.StateCommitment) {
			assert.Equal(t, mocks.GenericCommit(0), finalized)
		}

		tr, st := baselineFSM(t, StatusIndexed)
		st.forest = forest
		tr.index = index

		err := tr.ForwardHeight(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusForwarded, st.status)
		assert.Equal(t, mocks.GenericHeight+1, st.height)

		// Reset status to allow next call.
		st.status = StatusIndexed
		err = tr.ForwardHeight(st)

		require.NoError(t, err)
		assert.Equal(t, StatusForwarded, st.status)
		assert.Equal(t, mocks.GenericHeight+2, st.height)

		// First should have been called only once.
		assert.Equal(t, 1, firstCalled)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusEmpty)

		err := tr.ForwardHeight(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on first", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.FirstFunc = func(uint64) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndexed)
		tr.index = index

		err := tr.ForwardHeight(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on last", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.LastFunc = func(uint64) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndexed)
		tr.index = index

		err := tr.ForwardHeight(st)

		assert.Error(t, err)
	})
}

func TestTransitions_IndexChain(t *testing.T) {
	t.Run("nominal case index all", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}
		chain.CommitFunc = func(height uint64) (flow.StateCommitment, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericCommit(0), nil
		}
		chain.CollectionsFunc = func(height uint64) ([]*flow.LightCollection, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericCollections(2), nil
		}
		chain.GuaranteesFunc = func(height uint64) ([]*flow.CollectionGuarantee, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericGuarantees(2), nil
		}
		chain.TransactionsFunc = func(height uint64) ([]*flow.TransactionBody, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericTransactions(4), nil
		}
		chain.ResultsFunc = func(height uint64) ([]*flow.TransactionResult, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericResults(4), nil
		}
		chain.EventsFunc = func(height uint64) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericEvents(8), nil
		}
		chain.SealsFunc = func(height uint64) ([]*flow.Seal, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericSeals(4), nil
		}

		index := mocks.BaselineWriter(t)
		index.HeaderFunc = func(height uint64, header *flow.Header) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericHeader, header)

			return nil
		}
		index.CommitFunc = func(height uint64, commit flow.StateCommitment) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericCommit(0), commit)

			return nil
		}
		index.HeightFunc = func(blockID flow.Identifier, height uint64) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericHeader.ID(), blockID)

			return nil
		}
		index.CollectionsFunc = func(height uint64, collections []*flow.LightCollection) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericCollections(2), collections)

			return nil
		}
		index.GuaranteesFunc = func(height uint64, guarantees []*flow.CollectionGuarantee) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericGuarantees(2), guarantees)

			return nil
		}
		index.TransactionsFunc = func(height uint64, transactions []*flow.TransactionBody) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericTransactions(4), transactions)

			return nil
		}
		index.ResultsFunc = func(results []*flow.TransactionResult) error {
			assert.Equal(t, mocks.GenericResults(4), results)

			return nil
		}
		index.EventsFunc = func(height uint64, events []flow.Event) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericEvents(8), events)

			return nil
		}
		index.SealsFunc = func(height uint64, seals []*flow.Seal) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericSeals(4), seals)

			return nil
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain
		tr.index = index

		err := tr.IndexChain(st)

		require.NoError(t, err)
		assert.Equal(t, StatusUpdating, st.status)
	})

	t.Run("nominal case index nothing", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.CommitFunc = func(height uint64) (flow.StateCommitment, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericCommit(0), nil
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain
		tr.cfg.IndexCommit = false
		tr.cfg.IndexHeader = false
		tr.cfg.IndexTransactions = false
		tr.cfg.IndexCollections = false
		tr.cfg.IndexEvents = false
		tr.cfg.IndexSeals = false

		err := tr.IndexChain(st)

		require.NoError(t, err)
		assert.Equal(t, StatusUpdating, st.status)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusEmpty)

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve commit", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return flow.DummyStateCommitment, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write commit", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.CommitFunc = func(uint64, flow.StateCommitment) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve header", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.HeaderFunc = func(uint64, *flow.Header) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve transactions", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.TransactionsFunc = func(uint64) ([]*flow.TransactionBody, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve transaction results", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.ResultsFunc = func(uint64) ([]*flow.TransactionResult, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write transactions", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.ResultsFunc = func([]*flow.TransactionResult) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve collections", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.CollectionsFunc = func(uint64) ([]*flow.LightCollection, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write collections", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.CollectionsFunc = func(uint64, []*flow.LightCollection) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve guarantees", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.GuaranteesFunc = func(uint64) ([]*flow.CollectionGuarantee, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write guarantees", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.GuaranteesFunc = func(uint64, []*flow.CollectionGuarantee) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve events", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.EventsFunc = func(uint64) ([]flow.Event, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write events", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.EventsFunc = func(uint64, []flow.Event) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve seals", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.SealsFunc = func(uint64) ([]*flow.Seal, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write seals", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineWriter(t)
		index.SealsFunc = func(uint64, []*flow.Seal) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})
}

func baselineFSM(t *testing.T, status Status) (*Transitions, *State) {
	t.Helper()

	root := trie.NewEmptyMTrie()
	chain := mocks.BaselineChain(t)
	index := mocks.BaselineWriter(t)
	feeder := mocks.BaselineFeeder(t)
	forest := mocks.BaselineForest(t, true)

	once := &sync.Once{}
	doneCh := make(chan struct{})

	tr := &Transitions{
		cfg: Config{
			IndexCommit:       true,
			IndexHeader:       true,
			IndexPayloads:     true,
			IndexCollections:  true,
			IndexGuarantees:   true,
			IndexTransactions: true,
			IndexResults:      true,
			IndexEvents:       true,
			IndexSeals:        true,
		},
		log:   mocks.NoopLogger,
		root:  root,
		chain: chain,
		feed:  feeder,
		index: index,
		once:  once,
	}

	st := &State{
		forest:    forest,
		status:    status,
		height:    mocks.GenericHeight,
		last:      mocks.GenericCommit(1),
		next:      mocks.GenericCommit(0),
		registers: make(map[ledger.Path]*ledger.Payload),
		done:      doneCh,
	}

	return tr, st
}
