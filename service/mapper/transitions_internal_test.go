package mapper

import (
	"math"
	"sync"
	"testing"

	"github.com/onflow/flow-go/ledger/common/testutils"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/utils/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestNewTransitions(t *testing.T) {
	t.Run("nominal case, without options", func(t *testing.T) {
		chain := mocks.BaselineChain(t)
		updates := mocks.BaselineParser(t)
		read := mocks.BaselineReader(t)
		write := mocks.BaselineWriter(t)

		tr := NewTransitions(mocks.NoopLogger, chain, updates, read, write)

		assert.NotNil(t, tr)
		assert.Equal(t, chain, tr.chain)
		assert.Equal(t, updates, tr.updates)
		assert.Equal(t, write, tr.write)
		assert.NotNil(t, tr.once)
		assert.Equal(t, DefaultConfig, tr.cfg)
	})

	t.Run("nominal case, with option", func(t *testing.T) {
		chain := mocks.BaselineChain(t)
		updates := mocks.BaselineParser(t)
		read := mocks.BaselineReader(t)
		write := mocks.BaselineWriter(t)

		skip := true
		tr := NewTransitions(mocks.NoopLogger, chain, updates, read, write,
			WithSkipRegisters(skip),
		)

		assert.NotNil(t, tr)
		assert.Equal(t, chain, tr.chain)
		assert.Equal(t, updates, tr.updates)
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
		unittest.RunWithTempDir(t, func(dir string) {
			logger := unittest.Logger()
			tries := createSimpleTrie(t)
			fileName := "test_checkpoint_file"
			require.NoErrorf(t, wal.StoreCheckpointV6Concurrently(tries, dir, fileName, &logger), "fail to store checkpoint")
			tr, st := baselineFSM(t, StatusBootstrap)
			st.checkpointFileName = fileName
			st.checkpointDir = dir
			require.NoError(t, tr.BootstrapState(st))
		})
	})

	t.Run("check last height db updates", func(t *testing.T) {
		t.Parallel()
		unittest.RunWithTempDir(t, func(dir string) {
			logger := unittest.Logger()
			tries := createSimpleTrie(t)
			fileName := "test_checkpoint_file"
			require.NoErrorf(t, wal.StoreCheckpointV6Concurrently(tries, dir, fileName, &logger), "fail to store checkpoint")
			sporkHeight := uint64(10)
			writer := mocks.BaselineWriter(t)
			// spork height gets written as last height
			writer.LastFunc = func(height uint64) error {
				assert.Equal(t, height, sporkHeight)
				return nil
			}
			root := mocks.BaselineChain(t)
			root.RootFunc = func() (uint64, error) {
				return uint64(sporkHeight), nil
			}
			tr, st := baselineFSM(t, StatusBootstrap, withWriter(writer), withChain(root))
			st.checkpointFileName = fileName
			st.checkpointDir = dir
			require.NoError(t, tr.BootstrapState(st))
		})
	})

	t.Run("multiple batchs", func(t *testing.T) {
		t.Parallel()
		unittest.RunWithTempDir(t, func(dir string) {
			logger := unittest.Logger()
			trie1 := createTrieWithNPayloads(t, 3001) // 3001 payloads would require 4 batchs
			tries := []*trie.MTrie{trie1}
			fileName := "test_checkpoint_file"
			require.NoErrorf(t, wal.StoreCheckpointV6Concurrently(tries, dir, fileName, &logger), "fail to store checkpoint")
			tr, st := baselineFSM(t, StatusBootstrap)
			st.checkpointFileName = fileName
			st.checkpointDir = dir
			require.NoError(t, tr.BootstrapState(st))
		})
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
	trieUpdates := mocks.GenericTrieUpdates(5)
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdate)
		tu := mocks.GenericTrieUpdates(3)
		updates := mocks.BaselineParser(t)
		updates.UpdatesFunc = func() ([]*ledger.TrieUpdate, error) {
			return tu, nil
		}
		tr.updates = updates

		err := tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, tu, st.updates)
		// moved on to the next state
		assert.Equal(t, StatusCollect, st.status)
	})

	t.Run("nominal case with no available update temporarily", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusUpdate)

		// Set up the mock triereader to return an unavailable error on the first call and return successfully
		// to subsequent calls.
		var updateCalled bool
		updates := mocks.BaselineParser(t)
		updates.UpdatesFunc = func() ([]*ledger.TrieUpdate, error) {
			if !updateCalled {
				updateCalled = true
				return nil, archive.ErrUnavailable
			}
			return trieUpdates, nil
		}
		tr.updates = updates

		// The first call should not error but should not change the status of the FSM to updating. It should
		// instead remain Updating until a match is found.
		err := tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, StatusUpdate, st.status)

		// The second call is now successful and matches.
		err = tr.UpdateTree(st)

		require.NoError(t, err)
		assert.Equal(t, StatusCollect, st.status)
		assert.Equal(t, trieUpdates, st.updates)
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

	t.Run("handles triereader update failure", func(t *testing.T) {
		t.Parallel()

		updates := mocks.BaselineParser(t)
		updates.UpdatesFunc = func() ([]*ledger.TrieUpdate, error) {
			return nil, mocks.GenericError
		}

		tr, st := baselineFSM(t, StatusUpdate)
		tr.updates = updates

		err := tr.UpdateTree(st)

		assert.Error(t, err)
	})
}

func TestTransitions_CollectRegisters(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()
		tr, st := baselineFSM(t, StatusCollect)
		st.updates = mocks.GenericTrieUpdates(5)

		err := tr.CollectRegisters(st)

		require.NoError(t, err)
		assert.Equal(t, StatusMap, st.status)
		for _, wantPath := range mocks.GenericLedgerPaths(5) {
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

	t.Run("no updates empty trie", func(t *testing.T) {
		t.Parallel()
		tr, st := baselineFSM(t, StatusCollect)
		st.updates = make([]*ledger.TrieUpdate, 0)

		err := tr.CollectRegisters(st)

		require.NoError(t, err)
		assert.Equal(t, StatusMap, st.status)
		assert.Len(t, st.updates, 0)
	})

	t.Run("bootstrap case", func(t *testing.T) {
		t.Parallel()
		tr, st := baselineFSM(t, StatusCollect)
		updates := mocks.GenericTrieUpdates(5)
		// write ahead to registers, just like in the bootstrap
		for _, update := range updates {
			for i, path := range update.Paths {
				st.registers[path] = update.Payloads[i]
			}
		}

		// should be idempotent
		err := tr.CollectRegisters(st)

		require.NoError(t, err)
		assert.Equal(t, StatusMap, st.status)
		for _, wantPath := range mocks.GenericLedgerPaths(5) {
			assert.Contains(t, st.registers, wantPath)
		}
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
			mocks.GenericLedgerPath(4): mocks.GenericLedgerPayload(4),
			mocks.GenericLedgerPath(5): mocks.GenericLedgerPayload(5),
		}

		write := mocks.BaselineWriter(t)
		write.PayloadsFunc = func(height uint64, value []*ledger.Payload) error {
			assert.Equal(t, mocks.GenericHeight, height)

			// Expect the 5 entries from the map.
			assert.Len(t, value, 5)
			return nil
		}

		tr, st := baselineFSM(t, StatusMap)
		tr.write = write
		st.registers = testRegisters

		err := tr.MapRegisters(st)

		require.NoError(t, err)

		// Should be StatusForward because registers map was written
		assert.Empty(t, st.registers)
		assert.Equal(t, StatusForward, st.status)
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
		write.PayloadsFunc = func(uint64, []*ledger.Payload) error { return mocks.GenericError }

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
		tr, st := baselineFSM(t, StatusForward)
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
		t.Skip("will need to provide a mock of a reader that returns not found when querying last indexed height")

		tr, st := baselineFSM(t, StatusInitialize)

		// tr.cfg.BootstrapState = true

		err := tr.InitializeMapper(st)

		require.NoError(t, err)
		assert.Equal(t, StatusBootstrap, st.status)
	})

	t.Run("switches state to StatusResume if no bootstrapping configured", func(t *testing.T) {
		t.Parallel()

		tr, st := baselineFSM(t, StatusInitialize)

		// tr.cfg.BootstrapState = false

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
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		require.NoError(t, err)
		assert.Equal(t, StatusIndex, st.status)
		assert.Equal(t, header.Height+1, st.height)
	})

	t.Run("handles chain failure on Root", func(t *testing.T) {
		t.Parallel()

		chain := mocks.BaselineChain(t)
		chain.RootFunc = func() (uint64, error) {
			return 0, mocks.GenericError
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
			withChain(chain),
		)

		err := tr.ResumeIndexing(st)

		assert.Error(t, err)
	})
}

func createSimpleTrie(t *testing.T) []*trie.MTrie {
	emptyTrie := trie.NewEmptyMTrie()

	p1 := testutils.PathByUint8(0)
	v1 := testutils.LightPayload8('A', 'a')

	p2 := testutils.PathByUint8(1)
	v2 := testutils.LightPayload8('B', 'b')

	paths := []ledger.Path{p1, p2}
	payloads := []ledger.Payload{*v1, *v2}

	updatedTrie, _, err := trie.NewTrieWithUpdatedRegisters(emptyTrie, paths, payloads, true)
	require.NoError(t, err)
	tries := []*trie.MTrie{emptyTrie, updatedTrie}
	return tries
}

func createTrieWithNPayloads(t *testing.T, n int) *trie.MTrie {
	require.True(t, n <= math.MaxUint16, "invalid n")

	emptyTrie := trie.NewEmptyMTrie()

	paths := make([]ledger.Path, 0, n)
	payloads := make([]ledger.Payload, 0, n)

	for i := 0; i < n; i++ {
		p := testutils.PathByUint16(uint16(i))
		v := testutils.LightPayload8('A', 'a')

		paths = append(paths, p)
		payloads = append(payloads, *v)

	}
	updatedTrie, _, err := trie.NewTrieWithUpdatedRegisters(emptyTrie, paths, payloads, true)
	require.NoError(t, err)
	return updatedTrie
}

func baselineFSM(t *testing.T, status Status, opts ...func(tr *Transitions)) (*Transitions, *State) {
	t.Helper()

	chain := mocks.BaselineChain(t)
	updates := mocks.BaselineParser(t)
	read := mocks.BaselineReader(t)
	write := mocks.BaselineWriter(t)

	once := &sync.Once{}
	doneCh := make(chan struct{})

	tr := Transitions{
		cfg: Config{
			SkipRegisters: false,
			WaitInterval:  0,
		},
		log:     mocks.NoopLogger,
		chain:   chain,
		updates: updates,
		read:    read,
		write:   write,
		once:    once,
	}

	for _, opt := range opts {
		opt(&tr)
	}

	st := State{
		status:    status,
		height:    mocks.GenericHeight,
		updates:   make([]*ledger.TrieUpdate, 0),
		registers: make(map[ledger.Path]*ledger.Payload),
		done:      doneCh,
	}

	return &tr, &st
}

func withChain(chain archive.Chain) func(*Transitions) {
	return func(tr *Transitions) {
		tr.chain = chain
	}
}

func withReader(read archive.Reader) func(*Transitions) {
	return func(tr *Transitions) {
		tr.read = read
	}
}

func withWriter(write archive.Writer) func(*Transitions) {
	return func(tr *Transitions) {
		tr.write = write
	}
}
