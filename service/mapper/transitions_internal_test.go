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

	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestNewTransitions(t *testing.T) {
	t.Run("nominal case, without options", func(t *testing.T) {
		load := mocks.BaselineLoader(t)
		chain := mocks.BaselineChain(t)
		feed := mocks.BaselineFeeder(t)
		read := mocks.BaselineReader(t)
		write := mocks.BaselineWriter(t)

		tr := NewTransitions(mocks.NoopLogger, load, chain, feed, read, write)

		assert.NotNil(t, tr)
		assert.Equal(t, chain, tr.chain)
		assert.Equal(t, feed, tr.feed)
		assert.Equal(t, write, tr.write)
		assert.NotNil(t, tr.once)
		assert.Equal(t, DefaultConfig, tr.cfg)
	})

	t.Run("nominal case, with option", func(t *testing.T) {
		load := mocks.BaselineLoader(t)
		chain := mocks.BaselineChain(t)
		feed := mocks.BaselineFeeder(t)
		read := mocks.BaselineReader(t)
		write := mocks.BaselineWriter(t)

		skip := true
		tr := NewTransitions(mocks.NoopLogger, load, chain, feed, read, write,
			WithSkipRegisters(skip),
		)

		assert.NotNil(t, tr)
		assert.Equal(t, chain, tr.chain)
		assert.Equal(t, feed, tr.feed)
		assert.Equal(t, write, tr.write)
		assert.NotNil(t, tr.once)

		assert.NotEqual(t, DefaultConfig, tr.cfg)
		assert.Equal(t, skip, tr.cfg.SkipRegisters)
		assert.Equal(t, DefaultConfig.WaitInterval, tr.cfg.WaitInterval)
	})
}

func TestTransitions_BootstrapState(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusBootstrap)

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

		tr, st := baselineFSM(t, StatusForward)

		err := tr.BootstrapState(st)
		assert.Error(t, err)
	})

	t.Run("handles failure to get root height", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusBootstrap)

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		tr.chain = chain

		err := tr.BootstrapState(st)
		assert.Error(t, err)
	})
}

func TestTransitions_IndexChain(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
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

		write := mocks.BaselineWriter(t)
		write.HeaderFunc = func(height uint64, header *flow.Header) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericHeader, header)

			return nil
		}
		write.CommitFunc = func(height uint64, commit flow.StateCommitment) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericCommit(0), commit)

			return nil
		}
		write.HeightFunc = func(blockID flow.Identifier, height uint64) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericHeader.ID(), blockID)

			return nil
		}
		write.CollectionsFunc = func(height uint64, collections []*flow.LightCollection) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericCollections(2), collections)

			return nil
		}
		write.GuaranteesFunc = func(height uint64, guarantees []*flow.CollectionGuarantee) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericGuarantees(2), guarantees)

			return nil
		}
		write.TransactionsFunc = func(height uint64, transactions []*flow.TransactionBody) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericTransactions(4), transactions)

			return nil
		}
		write.ResultsFunc = func(results []*flow.TransactionResult) error {
			assert.Equal(t, mocks.GenericResults(4), results)

			return nil
		}
		write.EventsFunc = func(height uint64, events []flow.Event) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericEvents(8), events)

			return nil
		}
		write.SealsFunc = func(height uint64, seals []*flow.Seal) error {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericSeals(4), seals)

			return nil
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.chain = chain
		tr.write = write

		err := tr.IndexChain(st)

		require.NoError(t, err)
		assert.Equal(t, StatusUpdate, st.status)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusBootstrap)

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve commit", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return flow.DummyStateCommitment, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure to index commit", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.CommitFunc = func(uint64, flow.StateCommitment) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.write = write

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve header", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure to index header", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.HeaderFunc = func(uint64, *flow.Header) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.write = write

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve transactions", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.TransactionsFunc = func(uint64) ([]*flow.TransactionBody, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
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

		tr, st := baselineFSM(t, StatusIndex)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure to index transactions", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.ResultsFunc = func([]*flow.TransactionResult) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.write = write

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve collections", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.CollectionsFunc = func(uint64) ([]*flow.LightCollection, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure to index collections", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.CollectionsFunc = func(uint64, []*flow.LightCollection) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.write = write

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve guarantees", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.GuaranteesFunc = func(uint64) ([]*flow.CollectionGuarantee, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure to index guarantees", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.GuaranteesFunc = func(uint64, []*flow.CollectionGuarantee) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.write = write

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve events", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.EventsFunc = func(uint64) ([]flow.Event, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure to index events", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.EventsFunc = func(uint64, []flow.Event) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.write = write

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve seals", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.SealsFunc = func(uint64) ([]*flow.Seal, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure to index seals", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.SealsFunc = func(uint64, []*flow.Seal) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusIndex)
		tr.write = write

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})
}

func TestTransitions_UpdateTree(t *testing.T) {
	update := mocks.GenericTrieUpdate(0)
	tree := mocks.GenericTrie

	t.Run("nominal case without match", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdate)

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
		assert.Equal(t, StatusUpdate, st.status)
	})

	t.Run("nominal case with no available update temporarily", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdate)

		// Set up the mock feeder to return an unavailable error on the first call and return successfully
		// to subsequent calls.
		var updateCalled bool
		feeder := mocks.BaselineFeeder(t)
		feeder.UpdateFunc = func() (*ledger.TrieUpdate, error) {
			if !updateCalled {
				updateCalled = true
				return nil, dps.ErrUnavailable
			}
			return mocks.GenericTrieUpdate(0), nil
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
		assert.Equal(t, StatusUpdate, st.status)

		// The second call is now successful and matches.
		err = tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, StatusCollect, st.status)
	})

	t.Run("nominal case with match", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdate)

		err := tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, StatusCollect, st.status)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusBootstrap)

		err := tr.UpdateTree(st)

		assert.Error(t, err)
	})

	t.Run("handles feeder update failure", func(t *testing.T) {
		t.Parallel()

		feed := mocks.BaselineFeeder(t)
		feed.UpdateFunc = func() (*ledger.TrieUpdate, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusUpdate)
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

		tr, st := baselineFSM(t, StatusUpdate)
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

		tr, st := baselineFSM(t, StatusCollect)
		st.forest = forest

		err := tr.CollectRegisters(st)

		require.NoError(t, err)
		assert.Equal(t, StatusMap, st.status)
		for _, wantPath := range mocks.GenericLedgerPaths(6) {
			assert.Contains(t, st.registers, wantPath)
		}
	})

	t.Run("indexing payloads disabled", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusCollect)
		tr.cfg.SkipRegisters = true

		err := tr.CollectRegisters(st)

		require.NoError(t, err)
		assert.Empty(t, st.registers)
		assert.Equal(t, StatusForward, st.status)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusBootstrap)

		err := tr.CollectRegisters(st)

		assert.Error(t, err)
		assert.Empty(t, st.registers)
	})

	t.Run("handles missing tree for commit", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusCollect)
		st.forest = mocks.BaselineForest(t, false)

		err := tr.CollectRegisters(st)

		assert.Error(t, err)
		assert.Empty(t, st.registers)
	})
}

func TestTransitions_MapRegisters(t *testing.T) {
	t.Run("nominal case with registers to write", func(t *testing.T) {
		t.Parallel()

		// Path 2 and 4 are the same so the map effectively contains 5 entries.
		testRegisters := map[ledger.Path]*ledger.Payload{
			mocks.GenericLedgerPath(0): mocks.GenericLedgerPayload(0),
			mocks.GenericLedgerPath(1): mocks.GenericLedgerPayload(1),
			mocks.GenericLedgerPath(2): mocks.GenericLedgerPayload(2),
			mocks.GenericLedgerPath(1): mocks.GenericLedgerPayload(3),
			mocks.GenericLedgerPath(4): mocks.GenericLedgerPayload(4),
			mocks.GenericLedgerPath(5): mocks.GenericLedgerPayload(5),
		}

		write := mocks.BaselineWriter(t)
		write.PayloadsFunc = func(height uint64, paths []ledger.Path, value []*ledger.Payload) error {
			assert.Equal(t, mocks.GenericHeight, height)

			// Expect the 5 entries from the map.
			assert.Len(t, paths, 5)
			assert.Len(t, value, 5)
			return nil
		}

		tr, st := baselineFSM(t, StatusMap)
		tr.write = write
		st.registers = testRegisters

		err := tr.MapRegisters(st)

		require.NoError(t, err)

		// Should not be StateIndexed because registers map was not empty.
		assert.Empty(t, st.registers)
		assert.Equal(t, StatusMap, st.status)
	})

	t.Run("nominal case no more registers left to write", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusMap)

		err := tr.MapRegisters(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusForward, st.status)
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

		tr, st := baselineFSM(t, StatusBootstrap)
		st.registers = testRegisters

		err := tr.MapRegisters(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure", func(t *testing.T) {
		t.Parallel()

		testRegisters := map[ledger.Path]*ledger.Payload{
			mocks.GenericLedgerPath(0): mocks.GenericLedgerPayload(0),
			mocks.GenericLedgerPath(1): mocks.GenericLedgerPayload(1),
			mocks.GenericLedgerPath(2): mocks.GenericLedgerPayload(2),
			mocks.GenericLedgerPath(3): mocks.GenericLedgerPayload(3),
			mocks.GenericLedgerPath(4): mocks.GenericLedgerPayload(4),
			mocks.GenericLedgerPath(5): mocks.GenericLedgerPayload(5),
		}

		write := mocks.BaselineWriter(t)
		write.PayloadsFunc = func(uint64, []ledger.Path, []*ledger.Payload) error { return mocks.GenericError }

		tr, st := baselineFSM(t, StatusMap)
		tr.write = write
		st.registers = testRegisters

		err := tr.MapRegisters(st)

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
		write := mocks.BaselineWriter(t)
		write.FirstFunc = func(height uint64) error {
			assert.Equal(t, mocks.GenericHeight, height)
			firstCalled++
			return nil
		}
		write.LastFunc = func(height uint64) error {
			assert.Equal(t, mocks.GenericHeight+uint64(lastCalled), height)
			lastCalled++
			return nil
		}

		forest := mocks.BaselineForest(t, true)
		forest.ResetFunc = func(finalized flow.StateCommitment) {
			assert.Equal(t, mocks.GenericCommit(0), finalized)
		}

		tr, st := baselineFSM(t, StatusForward)
		st.forest = forest
		tr.write = write

		err := tr.ForwardHeight(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusIndex, st.status)
		assert.Equal(t, mocks.GenericHeight+1, st.height)

		// Reset status to allow next call.
		st.status = StatusForward
		err = tr.ForwardHeight(st)

		require.NoError(t, err)
		assert.Equal(t, StatusIndex, st.status)
		assert.Equal(t, mocks.GenericHeight+2, st.height)

		// First should have been called only once.
		assert.Equal(t, 1, firstCalled)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusBootstrap)

		err := tr.ForwardHeight(st)

		assert.Error(t, err)
	})

	t.Run("handles writer error on first", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.FirstFunc = func(uint64) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForward)
		tr.write = write

		err := tr.ForwardHeight(st)

		assert.Error(t, err)
	})

	t.Run("handles writer error on last", func(t *testing.T) {
		t.Parallel()

		write := mocks.BaselineWriter(t)
		write.LastFunc = func(uint64) error {
			return mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusForward)
		tr.write = write

		err := tr.ForwardHeight(st)

		assert.Error(t, err)
	})
}

func TestTransitions_InitializeMapper(t *testing.T) {
	t.Run("switches state to BootstrapState if configured to do so", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusInitialize)

		tr.cfg.BootstrapState = true

		err := tr.InitializeMapper(st)

		require.NoError(t, err)
		assert.Equal(t, StatusBootstrap, st.status)
	})

	t.Run("switches state to StatusResume if no bootstrapping configured", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusInitialize)

		tr.cfg.BootstrapState = false

		err := tr.InitializeMapper(st)

		require.NoError(t, err)
		assert.Equal(t, StatusResume, st.status)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusForward)

		err := tr.InitializeMapper(st)

		require.Error(t, err)
	})
}

func TestTransitions_ResumeIndexing(t *testing.T) {
	header := mocks.GenericHeader
	tree := mocks.GenericTrie
	commit := flow.StateCommitment(tree.RootHash())
	differentCommit := mocks.GenericCommit(0)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return header.Height, nil
		}

		writer := mocks.BaselineWriter(t)
		writer.FirstFunc = func(height uint64) error {
			assert.Equal(t, header.Height, height)

			return nil
		}

		loader := mocks.BaselineLoader(t)
		loader.TrieFunc = func() (*trie.MTrie, error) {
			return tree, nil
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}
		reader.CommitFunc = func(height uint64) (flow.StateCommitment, error) {
			assert.Equal(t, header.Height, height)

			return commit, nil
		}

		tr, st := baselineFSM(
			t,
			StatusResume,
			withReader(reader),
			withWriter(writer),
			withLoader(loader),
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		require.NoError(t, err)
		assert.Equal(t, StatusIndex, st.status)
		assert.Equal(t, header.Height+1, st.height)
		assert.Equal(t, flow.DummyStateCommitment, st.last)
		assert.Equal(t, commit, st.next)
	})

	t.Run("handles chain failure on Root", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		loader := mocks.BaselineLoader(t)
		loader.TrieFunc = func() (*trie.MTrie, error) {
			return tree, nil
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}
		reader.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return commit, nil
		}

		tr, st := baselineFSM(
			t,
			StatusResume,
			withReader(reader),
			withLoader(loader),
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		assert.Error(t, err)
	})

	t.Run("handles writer failure on First", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return header.Height, nil
		}

		writer := mocks.BaselineWriter(t)
		writer.FirstFunc = func(uint64) error {
			return mocks.GenericError
		}

		loader := mocks.BaselineLoader(t)
		loader.TrieFunc = func() (*trie.MTrie, error) {
			return tree, nil
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}
		reader.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return commit, nil
		}

		tr, st := baselineFSM(
			t,
			StatusResume,
			withWriter(writer),
			withReader(reader),
			withLoader(loader),
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		assert.Error(t, err)
	})

	t.Run("handles reader failure on Last", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return header.Height, nil
		}

		loader := mocks.BaselineLoader(t)
		loader.TrieFunc = func() (*trie.MTrie, error) {
			return tree, nil
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}
		reader.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return commit, nil
		}

		tr, st := baselineFSM(
			t,
			StatusResume,
			withReader(reader),
			withLoader(loader),
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		assert.Error(t, err)
	})

	t.Run("handles reader failure on Commit", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return header.Height, nil
		}

		loader := mocks.BaselineLoader(t)
		loader.TrieFunc = func() (*trie.MTrie, error) {
			return tree, nil
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}
		reader.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return flow.DummyStateCommitment, mocks.GenericError
		}

		tr, st := baselineFSM(
			t,
			StatusResume,
			withReader(reader),
			withLoader(loader),
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		assert.Error(t, err)
	})

	t.Run("handles loader failure on Trie", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return header.Height, nil
		}

		loader := mocks.BaselineLoader(t)
		loader.TrieFunc = func() (*trie.MTrie, error) {
			return nil, mocks.GenericError
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}
		reader.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return commit, nil
		}

		tr, st := baselineFSM(
			t,
			StatusResume,
			withReader(reader),
			withLoader(loader),
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		assert.Error(t, err)
	})

	t.Run("handles mismatch between tree root hash and indexed commit", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return header.Height, nil
		}

		loader := mocks.BaselineLoader(t)
		loader.TrieFunc = func() (*trie.MTrie, error) {
			return tree, nil
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}
		reader.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return differentCommit, nil
		}

		tr, st := baselineFSM(
			t,
			StatusResume,
			withReader(reader),
			withLoader(loader),
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		assert.Error(t, err)
	})

	t.Run("handles invalid status", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return header.Height, nil
		}

		loader := mocks.BaselineLoader(t)
		loader.TrieFunc = func() (*trie.MTrie, error) {
			return tree, nil
		}

		reader := mocks.BaselineReader(t)
		reader.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}
		reader.CommitFunc = func(uint64) (flow.StateCommitment, error) {
			return commit, nil
		}

		tr, st := baselineFSM(
			t,
			StatusForward,
			withReader(reader),
			withLoader(loader),
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		assert.Error(t, err)
	})
}

func baselineFSM(t *testing.T, status Status, opts ...func(tr *Transitions)) (*Transitions, *State) {
	t.Helper()

	load := mocks.BaselineLoader(t)
	chain := mocks.BaselineChain(t)
	feeder := mocks.BaselineFeeder(t)
	read := mocks.BaselineReader(t)
	write := mocks.BaselineWriter(t)
	forest := mocks.BaselineForest(t, true)

	once := &sync.Once{}
	doneCh := make(chan struct{})

	tr := Transitions{
		cfg: Config{
			BootstrapState: false,
			SkipRegisters:  false,
			WaitInterval:   0,
		},
		log:   mocks.NoopLogger,
		load:  load,
		chain: chain,
		feed:  feeder,
		read:  read,
		write: write,
		once:  once,
	}

	for _, opt := range opts {
		opt(&tr)
	}

	st := State{
		forest:    forest,
		status:    status,
		height:    mocks.GenericHeight,
		last:      mocks.GenericCommit(1),
		next:      mocks.GenericCommit(0),
		registers: make(map[ledger.Path]*ledger.Payload),
		done:      doneCh,
	}

	return &tr, &st
}

func withLoader(load Loader) func(*Transitions) {
	return func(tr *Transitions) {
		tr.load = load
	}
}

func withChain(chain dps.Chain) func(*Transitions) {
	return func(tr *Transitions) {
		tr.chain = chain
	}
}

func withFeeder(feed Feeder) func(*Transitions) {
	return func(tr *Transitions) {
		tr.feed = feed
	}
}

func withReader(read dps.Reader) func(*Transitions) {
	return func(tr *Transitions) {
		tr.read = read
	}
}

func withWriter(write dps.Writer) func(*Transitions) {
	return func(tr *Transitions) {
		tr.write = write
	}
}
