package chain_test

import (
	"math"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/service/chain"
)

const (
	testHeight               = 42
	testCommit               = "04bce5f43dc990800869fa05da95e92f4bac4b33a7579fcd778efc7b76a884f2"
	testChainID flow.ChainID = "flow-testnet"
)

var (
	testBlockID = flow.Identifier{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	testSealID  = flow.Identifier{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
)

func inMemoryChain(t *testing.T) *chain.ProtocolState {
	t.Helper()

	opts := badger.DefaultOptions("")
	opts.InMemory = true

	db, err := badger.Open(opts)
	require.NoError(t, err)

	err = db.Update(func(txn *badger.Txn) error {
		err = operation.InsertRootHeight(testHeight)(txn)
		if err != nil {
			return err
		}

		err = operation.InsertHeader(testBlockID, &flow.Header{ChainID: testChainID})(txn)
		if err != nil {
			return err
		}

		err = operation.IndexBlockHeight(testHeight, testBlockID)(txn)
		if err != nil {
			return err
		}

		err = operation.IndexBlockSeal(testBlockID, testSealID)(txn)
		if err != nil {
			return err
		}

		seal := &flow.Seal{
			FinalState: flow.StateCommitment(testCommit),
		}
		err = operation.InsertSeal(testSealID, seal)(txn)
		if err != nil {
			return err
		}

		events := []flow.Event{
			{
				Type:             "test",
				TransactionIndex: 1,
				EventIndex:       2,
			},
			{
				Type:             "test",
				TransactionIndex: 3,
				EventIndex:       4,
			},
		}
		err = operation.InsertEvent(testBlockID, events[0])(txn)
		if err != nil {
			return err
		}
		err = operation.InsertEvent(testBlockID, events[1])(txn)
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(t, err)

	return chain.FromProtocolState(db)
}

func TestProtocolState_Root(t *testing.T) {
	c := inMemoryChain(t)

	root, err := c.Root()
	assert.NoError(t, err)
	assert.Equal(t, uint64(testHeight), root)
}

func TestProtocolState_Header(t *testing.T) {
	c := inMemoryChain(t)

	header, err := c.Header(testHeight)
	assert.NoError(t, err)

	require.NotNil(t, header)
	assert.Equal(t, testChainID, header.ChainID)

	header, err = c.Header(math.MaxUint64)
	assert.Error(t, err)
}

func TestProtocolState_Commit(t *testing.T) {
	c := inMemoryChain(t)

	commit, err := c.Commit(testHeight)
	assert.NoError(t, err)
	assert.Equal(t, flow.StateCommitment(testCommit), commit)

	commit, err = c.Commit(math.MaxUint64)
	assert.Error(t, err)
}

func TestProtocolState_Events(t *testing.T) {
	c := inMemoryChain(t)

	events, err := c.Events(testHeight)
	assert.NoError(t, err)
	// TODO: Get a state with events to be able to test this.
	assert.Len(t, events, 2)

	_, err = c.Events(math.MaxUint64)
	assert.Error(t, err)
}
