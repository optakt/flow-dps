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
	"io"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/testing/mocks"
)

// For path/payload/trie/trie update test variables, see testing globals defined in pathings_test.go.
var (
	testHeight = uint64(42)
	testLog    = zerolog.New(io.Discard)
)

func TestNewTransitions(t *testing.T) {
	t.Run("nominal case, without options", func(t *testing.T) {
		load := &mocks.Loader{}
		chain := &mocks.Chain{}
		feed := &mocks.Feeder{}
		index := &mocks.Writer{}

		tr := NewTransitions(testLog, load, chain, feed, index)

		assert.NotNil(t, tr)
		assert.Equal(t, load, tr.load)
		assert.Equal(t, chain, tr.chain)
		assert.Equal(t, feed, tr.feed)
		assert.Equal(t, index, tr.index)
		assert.NotNil(t, tr.once)
		assert.Equal(t, DefaultConfig, tr.cfg)
	})

	t.Run("nominal case, with options", func(t *testing.T) {
		load := &mocks.Loader{}
		chain := &mocks.Chain{}
		feed := &mocks.Feeder{}
		index := &mocks.Writer{}

		tr := NewTransitions(testLog, load, chain, feed, index,
			WithIndexCommit(true),
			WithIndexHeader(true),
			WithIndexPayloads(true),
			WithIndexCollections(true),
			WithIndexTransactions(true),
		)

		assert.NotNil(t, tr)
		assert.Equal(t, load, tr.load)
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
		assert.True(t, tr.cfg.IndexTransactions)
	})
}

func TestTransitions_BootstrapState(t *testing.T) {

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusEmpty)

		// Copy state in local scope so that we can override its SaveFunc without impacting other
		// tests running in parallel.
		var saveCalled bool
		st.forest = &mocks.Forest{
			SaveFunc: func(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment) {
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
			},
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

		tr.chain = &mocks.Chain{
			RootFunc: func() (uint64, error) {
				return 0, mocks.DummyError
			},
		}

		err := tr.BootstrapState(st)
		assert.Error(t, err)
	})

	t.Run("handles failure to load checkpoint", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusEmpty)

		tr.load = &mocks.Loader{
			CheckpointFunc: func() (*trie.MTrie, error) {
				return nil, mocks.DummyError
			},
		}

		err := tr.BootstrapState(st)
		assert.Error(t, err)
	})
}

func TestTransitions_UpdateTree(t *testing.T) {
	t.Run("nominal case without match", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdating)

		testTrie, err := trie.NewMTrie(testRoot)
		require.NoError(t, err)

		forest := &mocks.Forest{
			SaveFunc: func(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment) {
				assert.NotZero(t, tree)

				// Expect the four deduplicated paths from testPaths.
				assert.Len(t, paths, 4)

				// Parent is empty since it is the empty trie.
				assert.Zero(t, parent)
			},
			TreeFunc: func(commit flow.StateCommitment) (*trie.MTrie, bool) {
				// Expect empty trie root hash as parent.
				assert.Zero(t, commit)

				return testTrie, true
			},
			HasFunc: func(_ flow.StateCommitment) bool {
				return false
			},
		}
		feed := &mocks.Feeder{
			UpdateFunc: func() (*ledger.TrieUpdate, error) {
				return testUpdate, nil
			},
		}

		tr.feed = feed
		st.forest = forest

		err = tr.UpdateTree(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusUpdating, st.status)
	})

	t.Run("nominal case with match", func(t *testing.T) {
		t.Parallel()

		forest := &mocks.Forest{
			HasFunc: func(_ flow.StateCommitment) bool {
				return true
			},
			SizeFunc: func() uint {
				return 42
			},
		}

		tr, st := baselineFSM(t, StatusUpdating)
		st.forest = forest

		err := tr.UpdateTree(st)

		assert.NoError(t, err)
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

		forest := &mocks.Forest{
			HasFunc: func(_ flow.StateCommitment) bool {
				return false
			},
		}
		feed := &mocks.Feeder{
			UpdateFunc: func() (*ledger.TrieUpdate, error) {
				return nil, mocks.DummyError
			},
		}

		tr, st := baselineFSM(t, StatusUpdating)
		st.forest = forest
		tr.feed = feed

		err := tr.UpdateTree(st)

		assert.Error(t, err)
	})

	t.Run("handles forest parent tree not found", func(t *testing.T) {
		t.Parallel()

		forest := &mocks.Forest{
			TreeFunc: func(_ flow.StateCommitment) (*trie.MTrie, bool) {
				return nil, false
			},
			HasFunc: func(_ flow.StateCommitment) bool {
				return false
			},
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

		testNextCommit := flow.StateCommitment{
			0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
			0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
			0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
			0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
		}
		testLastCommit := flow.StateCommitment{
			0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b,
			0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b,
			0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b,
			0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b,
		}

		testTrie, err := trie.NewMTrie(testRoot)
		require.NoError(t, err)

		forest := &mocks.Forest{
			HasFunc: func(_ flow.StateCommitment) bool {
				return true
			},
			TreeFunc: func(_ flow.StateCommitment) (*trie.MTrie, bool) {
				return testTrie, true
			},
			PathsFunc: func(_ flow.StateCommitment) ([]ledger.Path, bool) {
				return []ledger.Path{testPath1, testPath2, testPath3, testPath4, testPath5, testPath6}, true
			},
			ParentFunc: func(commit flow.StateCommitment) (flow.StateCommitment, bool) {
				assert.Equal(t, testNextCommit, commit)

				return testLastCommit, true
			},
		}

		tr, st := baselineFSM(t, StatusMatched)

		st.forest = forest

		err = tr.CollectRegisters(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusCollected, st.status)
		for _, wantPath := range testPaths {
			assert.Contains(t, st.registers, wantPath)
		}
	})

	t.Run("indexing payloads disabled", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusMatched)
		tr.cfg.IndexPayloads = false

		err := tr.CollectRegisters(st)

		assert.NoError(t, err)
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

		forest := &mocks.Forest{
			HasFunc: func(_ flow.StateCommitment) bool {
				return false
			},
		}

		tr, st := baselineFSM(t, StatusMatched)
		st.forest = forest

		err := tr.CollectRegisters(st)

		assert.Error(t, err)
		assert.Empty(t, st.registers)
	})
}

func TestTransitions_IndexRegisters(t *testing.T) {
	t.Run("nominal case with registers to index", func(t *testing.T) {
		t.Parallel()

		index := &mocks.Writer{
			PayloadsFunc: func(height uint64, paths []ledger.Path, value []*ledger.Payload) error {
				assert.Equal(t, testHeight, height)
				assert.Len(t, paths, 5)
				assert.Len(t, value, 5)
				return nil
			},
		}

		tr, st := baselineFSM(t, StatusCollected)
		tr.index = index
		st.registers = map[ledger.Path]*ledger.Payload{
			testPath1: testPayload1,
			testPath2: testPayload2,
			testPath3: testPayload3,
			testPath4: testPayload4,
			testPath5: testPayload5,
			testPath6: testPayload6,
		}

		err := tr.IndexRegisters(st)

		assert.NoError(t, err)

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

		tr, st := baselineFSM(t, StatusEmpty)

		err := tr.IndexRegisters(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure", func(t *testing.T) {
		t.Parallel()

		index := &mocks.Writer{
			PayloadsFunc: func(height uint64, paths []ledger.Path, value []*ledger.Payload) error { return mocks.DummyError },
		}

		tr, st := baselineFSM(t, StatusCollected)
		tr.index = index
		st.registers = map[ledger.Path]*ledger.Payload{
			testPath1: testPayload1,
			testPath2: testPayload2,
			testPath3: testPayload3,
			testPath4: testPayload4,
			testPath5: testPayload5,
			testPath6: testPayload6,
		}

		err := tr.IndexRegisters(st)

		assert.Error(t, err)
	})

}

func TestTransitions_ForwardHeight(t *testing.T) {
	testCommit := flow.StateCommitment{
		0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
		0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
		0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
		0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
	}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var (
			firstCalled int
			lastCalled  int
		)
		index := &mocks.Writer{
			FirstFunc: func(height uint64) error {
				assert.Equal(t, testHeight, height)
				firstCalled++
				return nil
			},
			LastFunc: func(height uint64) error {
				assert.Equal(t, testHeight+uint64(lastCalled), height)
				lastCalled++
				return nil
			},
		}

		forest := &mocks.Forest{
			ResetFunc: func(finalized flow.StateCommitment) {
				assert.Equal(t, testCommit, finalized)
			},
		}

		tr, st := baselineFSM(t, StatusIndexed)
		st.forest = forest
		tr.index = index

		err := tr.ForwardHeight(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusForwarded, st.status)
		assert.Equal(t, testHeight+1, st.height)

		// Reset status to allow next call.
		st.status = StatusIndexed
		err = tr.ForwardHeight(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusForwarded, st.status)
		assert.Equal(t, testHeight+2, st.height)

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

		index := &mocks.Writer{
			FirstFunc: func(height uint64) error {
				return mocks.DummyError
			},
			LastFunc: func(height uint64) error {
				return nil
			},
		}

		forest := &mocks.Forest{
			ResetFunc: func(finalized flow.StateCommitment) {},
		}

		tr, st := baselineFSM(t, StatusIndexed)
		st.forest = forest
		tr.index = index

		err := tr.ForwardHeight(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on last", func(t *testing.T) {
		t.Parallel()

		index := &mocks.Writer{
			FirstFunc: func(height uint64) error {
				return nil
			},
			LastFunc: func(height uint64) error {
				return mocks.DummyError
			},
		}

		forest := &mocks.Forest{
			ResetFunc: func(finalized flow.StateCommitment) {},
		}

		tr, st := baselineFSM(t, StatusIndexed)
		st.forest = forest
		tr.index = index

		err := tr.ForwardHeight(st)

		assert.Error(t, err)
	})
}

func TestTransitions_IndexChain(t *testing.T) {
	testCommit := flow.StateCommitment{
		0x2a, 0x04, 0x51, 0x3c, 0xc3, 0xc9, 0xa7, 0xf2,
		0xec, 0x08, 0x93, 0x56, 0x5f, 0x52, 0xc2, 0x9e,
		0x19, 0xf5, 0x58, 0x88, 0x10, 0x11, 0xe1, 0x13,
		0x60, 0x43, 0x9e, 0x57, 0x60, 0x18, 0xe3, 0xde,
	}
	testBlockID := flow.Identifier{
		0xd5, 0xf5, 0x0b, 0xc1, 0x7b, 0xa1, 0xea, 0xad,
		0x83, 0x0c, 0x86, 0xac, 0xce, 0x64, 0x5c, 0xa6,
		0xc0, 0x9f, 0xf0, 0xfe, 0xc5, 0x1c, 0x76, 0x10,
		0x03, 0x1c, 0xb9, 0x99, 0xa5, 0xb0, 0xb3, 0x22,
	}
	testEvents := []flow.Event{
		{TransactionID: flow.Identifier{0x1, 0x2}},
		{TransactionID: flow.Identifier{0x3, 0x4}},
	}
	testHeader := &flow.Header{
		ChainID: dps.FlowTestnet,
		Height:  testHeight,
	}
	testTransaction := &flow.TransactionBody{
		ReferenceBlockID: testBlockID,
	}
	testTransactions := []*flow.TransactionBody{testTransaction}
	testCollections := []*flow.LightCollection{{Transactions: []flow.Identifier{testTransaction.ID()}}}

	t.Run("nominal case index all", func(t *testing.T) {
		t.Parallel()

		chain := &mocks.Chain{
			HeaderFunc: func(height uint64) (*flow.Header, error) {
				assert.Equal(t, testHeight, height)

				return testHeader, nil
			},
			CommitFunc: func(height uint64) (flow.StateCommitment, error) {
				assert.Equal(t, testHeight, height)

				return testCommit, nil
			},
			CollectionsFunc: func(height uint64) ([]*flow.LightCollection, error) {
				assert.Equal(t, testHeight, height)

				return testCollections, nil
			},
			TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) {
				assert.Equal(t, testHeight, height)

				return testTransactions, nil
			},
			EventsFunc: func(height uint64) ([]flow.Event, error) {
				assert.Equal(t, testHeight, height)

				return testEvents, nil
			},
		}

		index := &mocks.Writer{
			HeaderFunc: func(height uint64, header *flow.Header) error {
				assert.Equal(t, testHeight, height)
				assert.Equal(t, testHeader, header)

				return nil
			},
			CommitFunc: func(height uint64, commit flow.StateCommitment) error {
				assert.Equal(t, testHeight, height)
				assert.Equal(t, testCommit, commit)

				return nil
			},
			HeightFunc: func(blockID flow.Identifier, height uint64) error {
				assert.Equal(t, testHeight, height)
				assert.Equal(t, testBlockID, blockID)

				return nil
			},
			CollectionsFunc: func(height uint64, collections []*flow.LightCollection) error {
				assert.Equal(t, testHeight, height)
				assert.Equal(t, testCollections, collections)

				return nil
			},
			TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error {
				assert.Equal(t, testHeight, height)
				assert.Equal(t, testTransactions, transactions)

				return nil
			},
			EventsFunc: func(height uint64, events []flow.Event) error {
				assert.Equal(t, testHeight, height)
				assert.Equal(t, testEvents, events)

				return nil
			},
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain
		tr.index = index

		err := tr.IndexChain(st)

		assert.NoError(t, err)
		assert.Equal(t, StatusUpdating, st.status)
	})

	t.Run("nominal case index nothing", func(t *testing.T) {
		t.Parallel()

		chain := &mocks.Chain{
			CommitFunc: func(height uint64) (flow.StateCommitment, error) {
				assert.Equal(t, testHeight, height)

				return testCommit, nil
			},
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain
		tr.cfg.IndexCommit = false
		tr.cfg.IndexHeader = false
		tr.cfg.IndexTransactions = false
		tr.cfg.IndexCollections = false
		tr.cfg.IndexEvents = false

		err := tr.IndexChain(st)

		assert.NoError(t, err)
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

		chain := &mocks.Chain{
			HeaderFunc: func(height uint64) (*flow.Header, error) { return testHeader, nil },
			CommitFunc: func(height uint64) (flow.StateCommitment, error) {
				return flow.StateCommitment{}, mocks.DummyError
			},
			CollectionsFunc:  func(height uint64) ([]*flow.LightCollection, error) { return testCollections, nil },
			TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) { return testTransactions, nil },
			EventsFunc:       func(height uint64) ([]flow.Event, error) { return testEvents, nil },
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write commit", func(t *testing.T) {
		t.Parallel()

		index := &mocks.Writer{
			HeaderFunc: func(height uint64, header *flow.Header) error { return nil },
			CommitFunc: func(height uint64, commit flow.StateCommitment) error {
				return mocks.DummyError
			},
			HeightFunc:       func(blockID flow.Identifier, height uint64) error { return nil },
			CollectionsFunc:  func(height uint64, collections []*flow.LightCollection) error { return nil },
			TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error { return nil },
			EventsFunc:       func(height uint64, events []flow.Event) error { return nil },
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve header", func(t *testing.T) {
		t.Parallel()

		chain := &mocks.Chain{
			HeaderFunc: func(height uint64) (*flow.Header, error) {
				return nil, mocks.DummyError
			},
			CommitFunc:       func(height uint64) (flow.StateCommitment, error) { return testCommit, nil },
			CollectionsFunc:  func(height uint64) ([]*flow.LightCollection, error) { return testCollections, nil },
			TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) { return testTransactions, nil },
			EventsFunc:       func(height uint64) ([]flow.Event, error) { return testEvents, nil },
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write header", func(t *testing.T) {
		t.Parallel()

		index := &mocks.Writer{
			HeaderFunc: func(height uint64, header *flow.Header) error {
				return mocks.DummyError
			},
			CommitFunc:       func(height uint64, commit flow.StateCommitment) error { return nil },
			HeightFunc:       func(blockID flow.Identifier, height uint64) error { return nil },
			CollectionsFunc:  func(height uint64, collections []*flow.LightCollection) error { return nil },
			TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error { return nil },
			EventsFunc:       func(height uint64, events []flow.Event) error { return nil },
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve transactions", func(t *testing.T) {
		t.Parallel()

		chain := &mocks.Chain{
			HeaderFunc:      func(height uint64) (*flow.Header, error) { return testHeader, nil },
			CommitFunc:      func(height uint64) (flow.StateCommitment, error) { return testCommit, nil },
			CollectionsFunc: func(height uint64) ([]*flow.LightCollection, error) { return testCollections, nil },
			TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) {
				return nil, mocks.DummyError
			},
			EventsFunc: func(height uint64) ([]flow.Event, error) { return testEvents, nil },
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write transactions", func(t *testing.T) {
		t.Parallel()

		index := &mocks.Writer{
			HeaderFunc:      func(height uint64, header *flow.Header) error { return nil },
			CommitFunc:      func(height uint64, commit flow.StateCommitment) error { return nil },
			HeightFunc:      func(blockID flow.Identifier, height uint64) error { return nil },
			CollectionsFunc: func(height uint64, collections []*flow.LightCollection) error { return nil },
			TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error {
				return mocks.DummyError
			},
			EventsFunc: func(height uint64, events []flow.Event) error { return nil },
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve collections", func(t *testing.T) {
		t.Parallel()

		chain := &mocks.Chain{
			HeaderFunc: func(height uint64) (*flow.Header, error) { return testHeader, nil },
			CommitFunc: func(height uint64) (flow.StateCommitment, error) { return testCommit, nil },
			CollectionsFunc: func(height uint64) ([]*flow.LightCollection, error) {
				return nil, mocks.DummyError
			},
			TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) { return testTransactions, nil },
			EventsFunc:       func(height uint64) ([]flow.Event, error) { return testEvents, nil },
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write transactions", func(t *testing.T) {
		t.Parallel()

		index := &mocks.Writer{
			HeaderFunc: func(height uint64, header *flow.Header) error { return nil },
			CommitFunc: func(height uint64, commit flow.StateCommitment) error { return nil },
			HeightFunc: func(blockID flow.Identifier, height uint64) error { return nil },
			CollectionsFunc: func(height uint64, collections []*flow.LightCollection) error {
				return mocks.DummyError
			},
			TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error { return nil },
			EventsFunc:       func(height uint64, events []flow.Event) error { return nil },
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles chain failure to retrieve events", func(t *testing.T) {
		t.Parallel()

		chain := &mocks.Chain{
			HeaderFunc:       func(height uint64) (*flow.Header, error) { return testHeader, nil },
			CommitFunc:       func(height uint64) (flow.StateCommitment, error) { return testCommit, nil },
			CollectionsFunc:  func(height uint64) ([]*flow.LightCollection, error) { return testCollections, nil },
			TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) { return testTransactions, nil },
			EventsFunc: func(height uint64) ([]flow.Event, error) {
				return nil, mocks.DummyError
			},
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.chain = chain

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure to write events", func(t *testing.T) {
		t.Parallel()

		index := &mocks.Writer{
			HeaderFunc:       func(height uint64, header *flow.Header) error { return nil },
			CommitFunc:       func(height uint64, commit flow.StateCommitment) error { return nil },
			HeightFunc:       func(blockID flow.Identifier, height uint64) error { return nil },
			PayloadsFunc:     func(height uint64, paths []ledger.Path, value []*ledger.Payload) error { return nil },
			CollectionsFunc:  func(height uint64, collections []*flow.LightCollection) error { return nil },
			TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error { return nil },
			EventsFunc: func(height uint64, events []flow.Event) error {
				return mocks.DummyError
			},
		}

		tr, st := baselineFSM(t, StatusForwarded)
		tr.index = index

		err := tr.IndexChain(st)

		assert.Error(t, err)
	})
}

func baselineFSM(t *testing.T, status Status) (*Transitions, *State) {
	t.Helper()

	testNextCommit := flow.StateCommitment{
		0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
		0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
		0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
		0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
	}
	testLastCommit := flow.StateCommitment{
		0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b,
		0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b,
		0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b,
		0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b, 0x2b,
	}
	testEvents := []flow.Event{
		{TransactionID: flow.Identifier{0x1, 0x2}},
		{TransactionID: flow.Identifier{0x3, 0x4}},
	}
	testHeader := &flow.Header{
		ChainID: dps.FlowTestnet,
		Height:  testHeight,
	}
	testTransaction := &flow.TransactionBody{
		ReferenceBlockID:   flow.Identifier{},
		Script:             nil,
		Arguments:          nil,
		GasLimit:           0,
		ProposalKey:        flow.ProposalKey{},
		Payer:              flow.Address{},
		Authorizers:        nil,
		PayloadSignatures:  nil,
		EnvelopeSignatures: nil,
	}
	testTransactions := []*flow.TransactionBody{testTransaction}
	testCollections := []*flow.LightCollection{{Transactions: []flow.Identifier{testTransaction.ID()}}}

	testTrie, err := trie.NewMTrie(testRoot)
	require.NoError(t, err)

	load := &mocks.Loader{
		CheckpointFunc: func() (*trie.MTrie, error) {
			return testTrie, nil
		},
	}

	chain := &mocks.Chain{
		RootFunc: func() (uint64, error) {
			return testHeight, nil
		},
		HeaderFunc: func(height uint64) (*flow.Header, error) {
			return testHeader, nil
		},
		CommitFunc: func(height uint64) (flow.StateCommitment, error) {
			return testNextCommit, nil
		},
		CollectionsFunc: func(height uint64) ([]*flow.LightCollection, error) {
			return testCollections, nil
		},
		TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) {
			return testTransactions, nil
		},
		EventsFunc: func(height uint64) ([]flow.Event, error) {
			return testEvents, nil
		},
	}

	index := &mocks.Writer{
		FirstFunc: func(height uint64) error {
			return nil
		},
		LastFunc: func(height uint64) error {
			return nil
		},
		HeaderFunc: func(height uint64, header *flow.Header) error {
			return nil
		},
		CommitFunc: func(height uint64, commit flow.StateCommitment) error {
			return nil
		},
		PayloadsFunc: func(height uint64, paths []ledger.Path, value []*ledger.Payload) error {
			return nil
		},
		HeightFunc: func(blockID flow.Identifier, height uint64) error {
			return nil
		},
		CollectionsFunc: func(height uint64, collections []*flow.LightCollection) error {
			return nil
		},
		TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error {
			return nil
		},
		EventsFunc: func(height uint64, events []flow.Event) error {
			return nil
		},
	}

	feeder := &mocks.Feeder{
		UpdateFunc: func() (*ledger.TrieUpdate, error) {
			return testUpdate, nil
		},
	}

	once := &sync.Once{}

	forest := &mocks.Forest{
		SaveFunc: func(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment) {},
		HasFunc: func(commit flow.StateCommitment) bool {
			return true
		},
		TreeFunc: func(commit flow.StateCommitment) (*trie.MTrie, bool) {
			return testTrie, true
		},
		PathsFunc: func(commit flow.StateCommitment) ([]ledger.Path, bool) {
			return testPaths, true
		},
		ParentFunc: func(commit flow.StateCommitment) (flow.StateCommitment, bool) {
			return testLastCommit, true
		},
		ResetFunc: func(finalized flow.StateCommitment) {},
		SizeFunc: func() uint {
			return 42
		},
	}

	doneCh := make(chan struct{})

	tr := &Transitions{
		cfg: Config{
			IndexCommit:       true,
			IndexHeader:       true,
			IndexPayloads:     true,
			IndexCollections:  true,
			IndexTransactions: true,
			IndexEvents:       true,
		},
		log:   testLog,
		load:  load,
		chain: chain,
		feed:  feeder,
		index: index,
		once:  once,
	}

	st := &State{
		forest:    forest,
		status:    status,
		height:    testHeight,
		last:      testLastCommit,
		next:      testNextCommit,
		registers: make(map[ledger.Path]*ledger.Payload),
		done:      doneCh,
	}

	return tr, st
}
