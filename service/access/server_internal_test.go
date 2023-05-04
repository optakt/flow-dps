package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go/engine/common/rpc/convert"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/onflow/flow/protobuf/go/flow/entities"

	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestNewServer(t *testing.T) {
	index := mocks.BaselineReader(t)
	codec := mocks.BaselineCodec(t)
	invoker := mocks.BaselineInvoker(t)

	s := NewServer(index, codec, invoker)

	assert.NotNil(t, s)
	assert.NotNil(t, s.codec)
	assert.Equal(t, index, s.index)
	assert.Equal(t, codec, s.codec)
	assert.Equal(t, invoker, s.invoker)
}

func TestServer_Ping(t *testing.T) {
	s := baselineServer(t)

	req := &access.PingRequest{}
	_, err := s.Ping(context.Background(), req)

	assert.NoError(t, err)
}

func TestServer_GetTransaction(t *testing.T) {
	tx := mocks.GenericTransaction(0)
	txID := tx.ID()

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.TransactionFunc = func(txID flow.Identifier) (*flow.TransactionBody, error) {
			assert.Equal(t, tx.ID(), txID)

			return tx, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		resp, err := s.GetTransaction(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, tx.Arguments, resp.Transaction.Arguments)
		assert.Equal(t, tx.ReferenceBlockID[:], resp.Transaction.ReferenceBlockId)
	})

	t.Run("handles indexer error on transaction", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.TransactionFunc = func(txID flow.Identifier) (*flow.TransactionBody, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		_, err := s.GetTransaction(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetTransactionResult(t *testing.T) {
	header := mocks.GenericHeader
	blockID := header.ID()
	tx := mocks.GenericTransaction(0)
	txID := tx.ID()
	result := mocks.GenericResult(0)

	t.Run("nominal case with status sealed", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.ResultFunc = func(gotTxID flow.Identifier) (*flow.TransactionResult, error) {
			assert.Equal(t, txID, gotTxID)

			return result, nil
		}
		index.HeightForTransactionFunc = func(gotTxID flow.Identifier) (uint64, error) {
			assert.Equal(t, txID, gotTxID)

			return header.Height, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, header.Height, height)

			return header, nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, header.Height, height)
			assert.Empty(t, types)

			return mocks.GenericEvents(4), nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		resp, err := s.GetTransactionResult(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, result.ErrorMessage, resp.ErrorMessage)
		assert.Equal(t, blockID[:], resp.BlockId)
		assert.Equal(t, entities.TransactionStatus_SEALED, resp.Status)
		assert.Equal(t, uint32(1), resp.StatusCode)
		assert.Equal(t, convert.IdentifierToMessage(txID), resp.TransactionId)
		assert.Equal(t, header.Height, resp.BlockHeight)
	})

	t.Run("nominal case with status executed and an error message", func(t *testing.T) {
		t.Parallel()

		height := header.Height + 999
		failedResult := mocks.GenericResult(0)
		failedResult.ErrorMessage = "dummy error"

		index := mocks.BaselineReader(t)
		index.ResultFunc = func(gotTxID flow.Identifier) (*flow.TransactionResult, error) {
			assert.Equal(t, txID, gotTxID)

			return failedResult, nil
		}
		index.HeightForTransactionFunc = func(gotTxID flow.Identifier) (uint64, error) {
			assert.Equal(t, txID, gotTxID)

			return height, nil
		}
		index.HeaderFunc = func(gotHeight uint64) (*flow.Header, error) {
			assert.Equal(t, height, gotHeight)

			return header, nil
		}
		index.EventsFunc = func(gotHeight uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, height, gotHeight)
			assert.Empty(t, types)

			return mocks.GenericEvents(4), nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		resp, err := s.GetTransactionResult(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, failedResult.ErrorMessage, resp.ErrorMessage)
		assert.Equal(t, blockID[:], resp.BlockId)
		assert.Equal(t, entities.TransactionStatus_EXECUTED, resp.Status)
		assert.Equal(t, uint32(0), resp.StatusCode)
	})

	t.Run("handles indexer error on result", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.ResultFunc = func(txID flow.Identifier) (*flow.TransactionResult, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		_, err := s.GetTransactionResult(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on HeightForTransaction", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForTransactionFunc = func(flow.Identifier) (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		_, err := s.GetTransactionResult(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		_, err := s.GetTransactionResult(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on last", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		_, err := s.GetTransactionResult(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on events", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(uint64, ...flow.EventType) ([]flow.Event, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: txID[:]}
		_, err := s.GetTransactionResult(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetTransactionResultByIndex(t *testing.T) {
	blockID := mocks.GenericBlock.BlockID
	header := mocks.GenericHeader
	txResults := mocks.GenericResults(3)

	var txIDs []flow.Identifier
	txMap := make(map[flow.Identifier]*flow.TransactionResult)
	for _, tx := range txResults {
		txMap[tx.ID()] = tx
		txIDs = append(txIDs, tx.ID())
	}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			return header.Height, nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			return txIDs, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return header, nil
		}
		index.HeightForTransactionFunc = func(txID flow.Identifier) (uint64, error) {
			return header.Height, nil
		}
		index.ResultFunc = func(txID flow.Identifier) (*flow.TransactionResult, error) {
			return txMap[txID], nil
		}

		// need to improve mocking events so we can relate it to the transactions better
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return mocks.GenericEvents(5), nil
		}
		index.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionByIndexRequest{
			BlockId: convert.IdentifierToMessage(blockID),
			Index:   0,
		}

		resp, err := s.GetTransactionResultByIndex(context.Background(), req)
		require.NoError(t, err)

		assert.Equal(t, resp.TransactionId, convert.IdentifierToMessage(txResults[0].TransactionID))
		assert.Equal(t, resp.BlockHeight, header.Height)
	})
}

func TestServer_GetTransactionResultsByBlockID(t *testing.T) {
	blockID := mocks.GenericBlock.BlockID
	header := mocks.GenericHeader
	txResults := mocks.GenericResults(3)

	var txIDs []flow.Identifier
	txMap := make(map[flow.Identifier]*flow.TransactionResult)
	for _, tx := range txResults {
		txMap[tx.ID()] = tx
		txIDs = append(txIDs, tx.ID())
	}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			return header.Height, nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			return txIDs, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return header, nil
		}
		index.HeightForTransactionFunc = func(txID flow.Identifier) (uint64, error) {
			return header.Height, nil
		}
		index.ResultFunc = func(txID flow.Identifier) (*flow.TransactionResult, error) {
			return txMap[txID], nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return mocks.GenericEvents(5), nil
		}
		index.LastFunc = func() (uint64, error) {
			return header.Height, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionsByBlockIDRequest{
			BlockId: convert.IdentifierToMessage(blockID),
		}
		resp, err := s.GetTransactionResultsByBlockID(context.Background(), req)
		require.NoError(t, err)

		for i := 0; i < len(resp.TransactionResults); i++ {
			assert.Equal(t, resp.TransactionResults[i].BlockId, convert.IdentifierToMessage(blockID))
			assert.Equal(t, resp.TransactionResults[i].BlockHeight, header.Height)
			assert.Equal(t, resp.TransactionResults[i].TransactionId, convert.IdentifierToMessage(txResults[i].TransactionID))
		}
	})
}

func TestServer_GetTransactionsByBlockID(t *testing.T) {
	txs := mocks.GenericTransactions(3)
	blockID := mocks.GenericBlock.BlockID
	header := mocks.GenericHeader

	var txIDs []flow.Identifier
	txMap := make(map[flow.Identifier]*flow.TransactionBody)
	for _, tx := range txs {
		txMap[tx.ID()] = tx
		txIDs = append(txIDs, tx.ID())
	}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			return header.Height, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return header, nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			return txIDs, nil
		}
		index.TransactionFunc = func(txID flow.Identifier) (*flow.TransactionBody, error) {
			return txMap[txID], nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionsByBlockIDRequest{
			BlockId: convert.IdentifierToMessage(blockID),
		}
		resp, err := s.GetTransactionsByBlockID(context.Background(), req)

		require.NoError(t, err)

		// excludes the last transaction (system tx)
		for i := 0; i < len(resp.Transactions)-1; i++ {
			assert.Equal(t, resp.Transactions[i].ReferenceBlockId, convert.IdentifierToMessage(txs[i].ReferenceBlockID))
			assert.Equal(t, resp.Transactions[i].Arguments, txs[i].Arguments)
		}
	})
}

func TestServer_GetEventsForBlockIDs(t *testing.T) {
	header := mocks.GenericHeader
	events := mocks.GenericEvents(6)
	types := mocks.GenericEventTypes(1)
	blockIDs := mocks.GenericBlockIDs(4)
	blocks := map[flow.Identifier]uint64{
		blockIDs[0]: mocks.GenericHeight,
		blockIDs[1]: mocks.GenericHeight + 1,
		blockIDs[2]: mocks.GenericHeight + 2,
		blockIDs[3]: mocks.GenericHeight + 3,
	}

	t.Run("nominal case with event type", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Contains(t, blocks, blockID)

			return blocks[blockID], nil
		}
		index.EventsFunc = func(height uint64, gotTypes ...flow.EventType) ([]flow.Event, error) {
			assert.InDelta(t, header.Height, height, float64(len(blocks)))
			assert.Equal(t, types, gotTypes)

			return events, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.InDelta(t, header.Height, height, float64(len(blocks)))

			return header, nil
		}

		s := baselineServer(t)
		s.index = index

		var ids [][]byte
		for _, id := range blockIDs {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForBlockIDsRequest{
			Type:     string(types[0]),
			BlockIds: ids,
		}
		resp, err := s.GetEventsForBlockIDs(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Results, len(blockIDs))
		for _, block := range resp.Results {
			assert.Len(t, block.Events, len(events))
		}
	})

	t.Run("nominal case without event type", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Contains(t, blocks, blockID)

			return blocks[blockID], nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.InDelta(t, header.Height, height, float64(len(blocks)))
			assert.Empty(t, types)

			return events, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.InDelta(t, header.Height, height, float64(len(blocks)))

			return header, nil
		}

		s := baselineServer(t)
		s.index = index

		var ids [][]byte
		for _, id := range blockIDs {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForBlockIDsRequest{
			Type:     "",
			BlockIds: ids,
		}
		resp, err := s.GetEventsForBlockIDs(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Results, len(blockIDs))
		for _, block := range resp.Results {
			assert.Len(t, block.Events, len(events))
		}
	})

	t.Run("handles indexer error on HeightForBlock", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(flow.Identifier) (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		var ids [][]byte
		for _, id := range blockIDs {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForBlockIDsRequest{
			BlockIds: ids,
			Type:     string(mocks.GenericEventType(0)),
		}
		_, err := s.GetEventsForBlockIDs(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on Events", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(uint64, ...flow.EventType) ([]flow.Event, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		var ids [][]byte
		for _, id := range blockIDs {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForBlockIDsRequest{
			BlockIds: ids,
			Type:     string(mocks.GenericEventType(0)),
		}
		_, err := s.GetEventsForBlockIDs(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		var ids [][]byte
		for _, id := range blockIDs {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForBlockIDsRequest{
			BlockIds: ids,
			Type:     string(mocks.GenericEventType(0)),
		}
		_, err := s.GetEventsForBlockIDs(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetEventsForHeightRange(t *testing.T) {
	header := mocks.GenericHeader
	events := mocks.GenericEvents(6)
	types := mocks.GenericEventTypes(1)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(h uint64, gotTypes ...flow.EventType) ([]flow.Event, error) {
			// Expect height to be between GenericHeight and GenericHeight + 3 since there are four
			// given blockIDs.
			assert.InDelta(t, header.Height, h, 3)
			assert.Equal(t, types, gotTypes)

			return events, nil
		}
		index.HeaderFunc = func(h uint64) (*flow.Header, error) {
			// Expect height to be between GenericHeight and GenericHeight + 3 since there are four
			// given blockIDs.
			assert.InDelta(t, header.Height, h, 3)

			return header, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: header.Height,
			EndHeight:   header.Height + 3,
			Type:        string(types[0]),
		}
		resp, err := s.GetEventsForHeightRange(context.Background(), req)

		require.NoError(t, err)
		assert.Len(t, resp.Results, 4)
		for _, block := range resp.Results {
			assert.Len(t, block.Events, len(events))
		}
	})

	t.Run("nominal case without event type", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(h uint64, types ...flow.EventType) ([]flow.Event, error) {
			// Expect height to be between GenericHeight and GenericHeight + 3 since there are four
			// given blockIDs.
			assert.InDelta(t, header.Height, h, 3)
			assert.Empty(t, types)

			return events, nil
		}
		index.HeaderFunc = func(h uint64) (*flow.Header, error) {
			// Expect height to be between GenericHeight and GenericHeight + 3 since there are four
			// given blockIDs.
			assert.InDelta(t, header.Height, h, 3)

			return header, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: header.Height,
			EndHeight:   header.Height + 3,
			Type:        "",
		}
		resp, err := s.GetEventsForHeightRange(context.Background(), req)

		assert.NoError(t, err)

		assert.Len(t, resp.Results, 4)
		for _, block := range resp.Results {
			assert.Len(t, block.Events, len(events))
		}
	})

	t.Run("handles indexer error on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: header.Height,
			EndHeight:   header.Height + 3,
			Type:        string(types[0]),
		}
		_, err := s.GetEventsForHeightRange(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on Events", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(uint64, ...flow.EventType) ([]flow.Event, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: header.Height,
			EndHeight:   header.Height + 3,
			Type:        string(types[0]),
		}
		_, err := s.GetEventsForHeightRange(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetNetworkParameters(t *testing.T) {
	header := mocks.GenericHeader

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, header.Height, mocks.GenericHeight)

			return header, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetNetworkParametersRequest{}
		resp, err := s.GetNetworkParameters(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, archive.FlowTestnet.String(), resp.ChainId)
	})

	t.Run("handles indexer failure on first", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.FirstFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetNetworkParametersRequest{}
		_, err := s.GetNetworkParameters(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetNetworkParametersRequest{}
		_, err := s.GetNetworkParameters(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetCollectionByID(t *testing.T) {
	collection := mocks.GenericCollection(0)
	collID := collection.ID()

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.CollectionFunc = func(gotCollID flow.Identifier) (*flow.LightCollection, error) {
			assert.Equal(t, collID, gotCollID)

			return collection, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetCollectionByIDRequest{Id: collID[:]}
		resp, err := s.GetCollectionByID(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp.Collection)
		assert.Len(t, resp.Collection.TransactionIds, 2)
	})

	t.Run("handles indexer failure on collection", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.CollectionFunc = func(flow.Identifier) (*flow.LightCollection, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetCollectionByIDRequest{Id: collID[:]}
		_, err := s.GetCollectionByID(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetAccount(t *testing.T) {
	account := mocks.GenericAccount

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(height uint64, regs flow.RegisterIDs) ([]flow.RegisterValue, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericRegisterValues(4), nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.AccountFunc = func(height uint64, address flow.Address) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, account.Address, address)

			return &account, nil
		}

		s := baselineServer(t)
		s.index = index
		s.invoker = invoker

		req := &access.GetAccountRequest{Address: account.Address[:]}
		resp, err := s.GetAccount(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp.Account)
		assert.Equal(t, account.Address[:], resp.Account.Address)
		assert.Equal(t, account.Balance, resp.Account.Balance)
	})

	t.Run("handles indexer failure on Last", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetAccountRequest{Address: account.Address[:]}
		_, err := s.GetAccount(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles invoker failure on GetAccount", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.AccountFunc = func(uint64, flow.Address) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.GetAccountRequest{Address: account.Address[:]}
		_, err := s.GetAccount(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetAccountAtLatestBlock(t *testing.T) {
	account := mocks.GenericAccount

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(height uint64, regs flow.RegisterIDs) ([]flow.RegisterValue, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericRegisterValues(4), nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.AccountFunc = func(height uint64, address flow.Address) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, account.Address, address)

			return &account, nil
		}

		s := baselineServer(t)
		s.index = index
		s.invoker = invoker

		req := &access.GetAccountAtLatestBlockRequest{Address: account.Address[:]}
		resp, err := s.GetAccountAtLatestBlock(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp.Account)
		assert.Equal(t, account.Address[:], resp.Account.Address)
		assert.Equal(t, account.Balance, resp.Account.Balance)
	})

	t.Run("handles indexer failure on Last", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetAccountAtLatestBlockRequest{Address: account.Address[:]}
		_, err := s.GetAccountAtLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles invoker failure on GetAccount", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.AccountFunc = func(uint64, flow.Address) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.GetAccountAtLatestBlockRequest{Address: account.Address[:]}
		_, err := s.GetAccountAtLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetAccountAtBlockHeight(t *testing.T) {
	account := mocks.GenericAccount

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		height := mocks.GenericHeight + 999

		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(gotHeight uint64, regs flow.RegisterIDs) ([]flow.RegisterValue, error) {
			assert.Equal(t, height, gotHeight)

			return mocks.GenericRegisterValues(4), nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.AccountFunc = func(gotHeight uint64, address flow.Address) (*flow.Account, error) {
			assert.Equal(t, height, gotHeight)
			assert.Equal(t, account.Address, address)

			return &account, nil
		}

		s := baselineServer(t)
		s.index = index
		s.invoker = invoker

		req := &access.GetAccountAtBlockHeightRequest{
			BlockHeight: height,
			Address:     account.Address[:],
		}
		resp, err := s.GetAccountAtBlockHeight(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp.Account)
		assert.Equal(t, account.Address[:], resp.Account.Address)
		assert.Equal(t, account.Balance, resp.Account.Balance)
	})

	t.Run("handles invoker failure on GetAccount", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.AccountFunc = func(uint64, flow.Address) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.GetAccountAtBlockHeightRequest{
			BlockHeight: mocks.GenericHeight,
			Address:     account.Address[:],
		}
		_, err := s.GetAccountAtBlockHeight(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_ExecuteScriptAtBlockHeight(t *testing.T) {
	cadenceValue := cadence.NewUInt64(mocks.GenericHeight)
	cadenceValueBytes, err := json.Encode(cadenceValue)
	require.NoError(t, err)

	genericAmountBytes, err := json.Encode(mocks.GenericAmount(0))
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericBytes, script)
			assert.Equal(t, []cadence.Value{cadenceValue}, parameters)

			return mocks.GenericAmount(0), nil
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.ExecuteScriptAtBlockHeightRequest{
			BlockHeight: mocks.GenericHeight,
			Script:      mocks.GenericBytes,
			Arguments:   [][]byte{cadenceValueBytes},
		}
		resp, err := s.ExecuteScriptAtBlockHeight(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, genericAmountBytes, resp.Value)
	})

	t.Run("handles invoker failure", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(uint64, []byte, []cadence.Value) (cadence.Value, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.ExecuteScriptAtBlockHeightRequest{
			BlockHeight: mocks.GenericHeight,
			Script:      mocks.GenericBytes,
			Arguments:   [][]byte{cadenceValueBytes},
		}
		_, err = s.ExecuteScriptAtBlockHeight(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_ExecuteScriptAtBlockID(t *testing.T) {
	blockID := mocks.GenericHeader.ID()

	cadenceValue := cadence.NewUInt64(mocks.GenericHeight)
	cadenceValueBytes, err := json.Encode(cadenceValue)
	require.NoError(t, err)

	genericAmountBytes, err := json.Encode(mocks.GenericAmount(0))
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericBytes, script)
			assert.Equal(t, []cadence.Value{cadenceValue}, parameters)

			return mocks.GenericAmount(0), nil
		}

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(got flow.Identifier) (uint64, error) {
			assert.Equal(t, blockID, got)

			return mocks.GenericHeight, nil
		}

		s := baselineServer(t)
		s.index = index
		s.invoker = invoker

		req := &access.ExecuteScriptAtBlockIDRequest{
			BlockId:   blockID[:],
			Script:    mocks.GenericBytes,
			Arguments: [][]byte{cadenceValueBytes},
		}
		resp, err := s.ExecuteScriptAtBlockID(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, genericAmountBytes, resp.Value)
	})

	t.Run("handles index failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(flow.Identifier) (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.ExecuteScriptAtBlockIDRequest{
			BlockId:   blockID[:],
			Script:    mocks.GenericBytes,
			Arguments: [][]byte{cadenceValueBytes},
		}
		_, err = s.ExecuteScriptAtBlockID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles invoker failure", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(uint64, []byte, []cadence.Value) (cadence.Value, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.ExecuteScriptAtBlockIDRequest{
			BlockId:   blockID[:],
			Script:    mocks.GenericBytes,
			Arguments: [][]byte{cadenceValueBytes},
		}
		_, err = s.ExecuteScriptAtBlockID(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_ExecuteScriptAtLatestBlock(t *testing.T) {
	cadenceValue := cadence.NewUInt64(mocks.GenericHeight)
	cadenceValueBytes, err := json.Encode(cadenceValue)
	require.NoError(t, err)

	genericAmountBytes, err := json.Encode(mocks.GenericAmount(0))
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, mocks.GenericBytes, script)
			assert.Equal(t, []cadence.Value{cadenceValue}, parameters)

			return mocks.GenericAmount(0), nil
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.ExecuteScriptAtLatestBlockRequest{
			Script:    mocks.GenericBytes,
			Arguments: [][]byte{cadenceValueBytes},
		}
		resp, err := s.ExecuteScriptAtLatestBlock(context.Background(), req)

		require.NoError(t, err)

		assert.Equal(t, genericAmountBytes, resp.Value)
	})

	t.Run("handles index failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.ExecuteScriptAtLatestBlockRequest{
			Script:    mocks.GenericBytes,
			Arguments: [][]byte{cadenceValueBytes},
		}
		_, err = s.ExecuteScriptAtLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles invoker failure", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(uint64, []byte, []cadence.Value) (cadence.Value, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.ExecuteScriptAtLatestBlockRequest{
			Script:    mocks.GenericBytes,
			Arguments: [][]byte{cadenceValueBytes},
		}
		_, err = s.ExecuteScriptAtLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetLatestBlock(t *testing.T) {
	header := mocks.GenericHeader
	blockID := header.ID()
	sealIDs := mocks.GenericSealIDs(6)
	collIDs := mocks.GenericCollectionIDs(6)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var sealCalled, collCalled int
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(id flow.Identifier) (uint64, error) {
			assert.Equal(t, blockID, id)

			return header.Height, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, header.Height, height)

			return header, nil
		}
		index.SealsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, header.Height, height)

			return sealIDs, nil
		}
		index.SealFunc = func(sealID flow.Identifier) (*flow.Seal, error) {
			assert.Contains(t, sealIDs, sealID)

			seal := mocks.GenericSeal(sealCalled)
			sealCalled++

			return seal, nil
		}
		index.CollectionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, header.Height, height)

			return collIDs, nil
		}
		index.CollectionFunc = func(collID flow.Identifier) (*flow.LightCollection, error) {
			assert.Contains(t, collIDs, collID)

			collection := mocks.GenericCollection(collCalled)
			collCalled++

			return collection, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetLatestBlockRequest{}
		resp, err := s.GetLatestBlock(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, blockID[:], resp.Block.Id)
		assert.Equal(t, header.Height, resp.Block.Height)
		assert.Equal(t, header.ParentID[:], resp.Block.ParentId)
		assert.NotZero(t, resp.Block.Timestamp)

		for i, collID := range collIDs {
			seal := mocks.GenericSeal(i)
			wantSeal := &entities.BlockSeal{
				BlockId:                    seal.BlockID[:],
				ExecutionReceiptId:         seal.ResultID[:],
				ExecutionReceiptSignatures: [][]byte{},
			}
			assert.Contains(t, resp.Block.BlockSeals, wantSeal)

			guarantee := mocks.GenericGuarantee(i)
			wantGuarantee := &entities.CollectionGuarantee{
				CollectionId: collID[:],
				Signatures:   [][]byte{guarantee.Signature},
			}
			assert.Contains(t, resp.Block.CollectionGuarantees, wantGuarantee)
		}
	})

	t.Run("handles indexer failure on Last", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetLatestBlockRequest{}
		_, err := s.GetLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetLatestBlockRequest{}
		_, err := s.GetLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on SealsByHeight", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.SealsByHeightFunc = func(uint64) ([]flow.Identifier, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetLatestBlockRequest{}
		_, err := s.GetLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on CollectionsByHeight", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.CollectionsByHeightFunc = func(uint64) ([]flow.Identifier, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetLatestBlockRequest{}
		_, err := s.GetLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetBlockByID(t *testing.T) {
	header := mocks.GenericHeader
	blockID := header.ID()
	sealIDs := mocks.GenericSealIDs(6)
	collIDs := mocks.GenericCollectionIDs(6)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var sealCalled, collCalled int
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(gotBlockID flow.Identifier) (uint64, error) {
			assert.Equal(t, blockID, gotBlockID)

			return mocks.GenericHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, header.Height, height)

			return mocks.GenericHeader, nil
		}
		index.SealsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, header.Height, height)

			return sealIDs, nil
		}
		index.SealFunc = func(sealID flow.Identifier) (*flow.Seal, error) {
			assert.Contains(t, sealIDs, sealID)

			seal := mocks.GenericSeal(sealCalled)
			sealCalled++

			return seal, nil
		}
		index.CollectionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, header.Height, height)

			return collIDs, nil
		}
		index.CollectionFunc = func(collID flow.Identifier) (*flow.LightCollection, error) {
			assert.Contains(t, collIDs, collID)

			collection := mocks.GenericCollection(collCalled)
			collCalled++

			return collection, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: blockID[:]}
		resp, err := s.GetBlockByID(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, blockID[:], resp.Block.Id)
		assert.Equal(t, header.Height, resp.Block.Height)
		assert.Equal(t, header.ParentID[:], resp.Block.ParentId)
		assert.NotZero(t, resp.Block.Timestamp)

		for i, collID := range collIDs {
			seal := mocks.GenericSeal(i)
			wantSeal := &entities.BlockSeal{
				BlockId:                    seal.BlockID[:],
				ExecutionReceiptId:         seal.ResultID[:],
				ExecutionReceiptSignatures: [][]byte{},
			}
			assert.Contains(t, resp.Block.BlockSeals, wantSeal)

			guarantee := mocks.GenericGuarantee(i)
			wantGuarantee := &entities.CollectionGuarantee{
				CollectionId: collID[:],
				Signatures:   [][]byte{guarantee.Signature},
			}
			assert.Contains(t, resp.Block.CollectionGuarantees, wantGuarantee)
		}
	})

	t.Run("handles indexer failure on HeightForBlock", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(flow.Identifier) (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: blockID[:]}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: blockID[:]}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on SealsByHeight", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.SealsByHeightFunc = func(uint64) ([]flow.Identifier, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: blockID[:]}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on Seal", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.SealFunc = func(flow.Identifier) (*flow.Seal, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: blockID[:]}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on CollectionsByHeight", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.CollectionsByHeightFunc = func(uint64) ([]flow.Identifier, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: blockID[:]}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on Guarantee", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.GuaranteeFunc = func(flow.Identifier) (*flow.CollectionGuarantee, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: blockID[:]}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetBlockByHeight(t *testing.T) {
	header := mocks.GenericHeader
	blockID := header.ID()
	sealIDs := mocks.GenericSealIDs(6)
	collIDs := mocks.GenericCollectionIDs(6)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var sealCalled, collCalled int
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(gotBlockID flow.Identifier) (uint64, error) {
			assert.Equal(t, blockID, gotBlockID)

			return header.Height, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, header.Height, height)

			return header, nil
		}
		index.SealsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, header.Height, height)

			return sealIDs, nil
		}
		index.SealFunc = func(sealID flow.Identifier) (*flow.Seal, error) {
			assert.Contains(t, sealIDs, sealID)

			seal := mocks.GenericSeal(sealCalled)
			sealCalled++

			return seal, nil
		}
		index.CollectionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, header.Height, height)

			return collIDs, nil
		}
		index.CollectionFunc = func(collID flow.Identifier) (*flow.LightCollection, error) {
			assert.Contains(t, collIDs, collID)

			collection := mocks.GenericCollection(collCalled)
			collCalled++

			return collection, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByHeightRequest{Height: header.Height}
		resp, err := s.GetBlockByHeight(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, blockID[:], resp.Block.Id)
		assert.Equal(t, mocks.GenericHeight, resp.Block.Height)
		assert.Equal(t, mocks.GenericHeader.ParentID[:], resp.Block.ParentId)
		assert.NotZero(t, resp.Block.Timestamp)

		for i, collID := range collIDs {
			seal := mocks.GenericSeal(i)
			wantSeal := &entities.BlockSeal{
				BlockId:                    seal.BlockID[:],
				ExecutionReceiptId:         seal.ResultID[:],
				ExecutionReceiptSignatures: [][]byte{},
			}
			assert.Contains(t, resp.Block.BlockSeals, wantSeal)

			guarantee := mocks.GenericGuarantee(i)
			wantGuarantee := &entities.CollectionGuarantee{
				CollectionId: collID[:],
				Signatures:   [][]byte{guarantee.Signature},
			}
			assert.Contains(t, resp.Block.CollectionGuarantees, wantGuarantee)
		}
	})

	t.Run("handles indexer failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByHeightRequest{Height: header.Height}
		_, err := s.GetBlockByHeight(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on SealsByHeight", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.SealsByHeightFunc = func(uint64) ([]flow.Identifier, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByHeightRequest{Height: header.Height}
		_, err := s.GetBlockByHeight(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on CollectionsByHeight", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.CollectionsByHeightFunc = func(uint64) ([]flow.Identifier, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByHeightRequest{Height: header.Height}
		_, err := s.GetBlockByHeight(context.Background(), req)

		assert.Error(t, err)
	})
}

func baselineServer(t *testing.T) *Server {
	t.Helper()

	s := Server{
		codec:   mocks.BaselineCodec(t),
		index:   mocks.BaselineReader(t),
		invoker: mocks.BaselineInvoker(t),
	}

	return &s
}
