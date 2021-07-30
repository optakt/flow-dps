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

package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/onflow/flow/protobuf/go/flow/entities"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/testing/mocks"
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

func TestServer_GetLatestBlockHeader(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		resp, err := s.GetLatestBlockHeader(context.Background(), nil)

		assert.NoError(t, err)

		assert.Equal(t, mocks.ByteSlice(mocks.GenericHeader.ID()), resp.Block.Id)
		assert.Equal(t, mocks.GenericHeader.Height, resp.Block.Height)
	})

	t.Run("handles indexer error on Last", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		_, err := s.GetLatestBlockHeader(context.Background(), nil)

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

		_, err := s.GetLatestBlockHeader(context.Background(), nil)

		assert.Error(t, err)
	})
}

func TestServer_GetBlockHeaderByID(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Equal(t, mocks.GenericIdentifier(0), blockID)

			return mocks.GenericHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockHeaderByIDRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
		resp, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.NoError(t, err)

		assert.Equal(t, mocks.ByteSlice(mocks.GenericHeader.ID()), resp.Block.Id)
		assert.Equal(t, mocks.GenericHeader.Height, resp.Block.Height)
	})

	t.Run("handles indexer error on HeightForBlock", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockHeaderByIDRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
		_, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			return mocks.GenericHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockHeaderByIDRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
		_, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetBlockHeaderByHeight(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockHeaderByHeightRequest{Height: mocks.GenericHeight}
		resp, err := s.GetBlockHeaderByHeight(context.Background(), req)

		assert.NoError(t, err)

		assert.Equal(t, mocks.ByteSlice(mocks.GenericHeader.ID()), resp.Block.Id)
		assert.Equal(t, mocks.GenericHeader.Height, resp.Block.Height)
	})

	t.Run("handles indexer error on header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockHeaderByHeightRequest{Height: mocks.GenericHeight}
		_, err := s.GetBlockHeaderByHeight(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetTransaction(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tx := mocks.GenericTransaction(0)

		index := mocks.BaselineReader(t)
		index.TransactionFunc = func(txID flow.Identifier) (*flow.TransactionBody, error) {
			assert.Equal(t, mocks.GenericIdentifier(0), txID)

			return tx, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetTransactionRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
		resp, err := s.GetTransaction(context.Background(), req)

		assert.NoError(t, err)

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

		req := &access.GetTransactionRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
		_, err := s.GetTransaction(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetTransactionResult(t *testing.T) {
	t.Run("nominal case with status sealed", func(t *testing.T) {
		t.Parallel()

		tx := mocks.GenericTransaction(0)
		result := mocks.GenericResult(0)

		index := mocks.BaselineReader(t)
		index.ResultFunc = func(txID flow.Identifier) (*flow.TransactionResult, error) {
			assert.Equal(t, tx.ID(), txID)

			return result, nil
		}
		index.HeightForTransactionFunc = func(txID flow.Identifier) (uint64, error) {
			assert.Equal(t, tx.ID(), txID)

			return mocks.GenericHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Empty(t, types)

			return mocks.GenericEvents(4), nil
		}

		s := baselineServer(t)
		s.index = index

		txID := tx.ID()
		req := &access.GetTransactionRequest{Id: txID[:]}
		resp, err := s.GetTransactionResult(context.Background(), req)

		assert.NoError(t, err)

		assert.Equal(t, result.ErrorMessage, resp.ErrorMessage)
		assert.Equal(t, mocks.ByteSlice(mocks.GenericHeader.ID()), resp.BlockId)
		assert.Equal(t, entities.TransactionStatus_SEALED, resp.Status)
		assert.Equal(t, uint32(1), resp.StatusCode)
	})

	t.Run("nominal case with status executed and an error message", func(t *testing.T) {
		t.Parallel()

		txHeight := mocks.GenericHeight + 999
		tx := mocks.GenericTransaction(0)
		result := mocks.GenericResult(0)
		result.ErrorMessage = "dummy error"

		index := mocks.BaselineReader(t)
		index.ResultFunc = func(txID flow.Identifier) (*flow.TransactionResult, error) {
			assert.Equal(t, tx.ID(), txID)

			return result, nil
		}
		index.HeightForTransactionFunc = func(txID flow.Identifier) (uint64, error) {
			assert.Equal(t, tx.ID(), txID)

			return txHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, txHeight, height)

			return mocks.GenericHeader, nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, txHeight, height)
			assert.Empty(t, types)

			return mocks.GenericEvents(4), nil
		}

		s := baselineServer(t)
		s.index = index

		txID := tx.ID()
		req := &access.GetTransactionRequest{Id: txID[:]}
		resp, err := s.GetTransactionResult(context.Background(), req)

		assert.NoError(t, err)

		assert.Equal(t, result.ErrorMessage, resp.ErrorMessage)
		assert.Equal(t, mocks.ByteSlice(mocks.GenericHeader.ID()), resp.BlockId)
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

		req := &access.GetTransactionRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
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

		req := &access.GetTransactionRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
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

		req := &access.GetTransactionRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
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

		req := &access.GetTransactionRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
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

		req := &access.GetTransactionRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
		_, err := s.GetTransactionResult(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetEventsForBlockIDs(t *testing.T) {
	t.Run("nominal case with event type", func(t *testing.T) {
		t.Parallel()

		events := mocks.GenericEvents(6)
		blockIDs := mocks.GenericIdentifiers(4)
		blocks := map[flow.Identifier]uint64{
			blockIDs[0]: mocks.GenericHeight,
			blockIDs[1]: mocks.GenericHeight + 1,
			blockIDs[2]: mocks.GenericHeight + 2,
			blockIDs[3]: mocks.GenericHeight + 3,
		}

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Contains(t, blocks, blockID)

			return blocks[blockID], nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.InDelta(t, mocks.GenericHeight, height, float64(len(blocks)))
			assert.Equal(t, mocks.GenericEventTypes(1), types)

			return events, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.InDelta(t, mocks.GenericHeight, height, float64(len(blocks)))

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		var ids [][]byte
		for _, id := range blockIDs {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForBlockIDsRequest{
			Type:     string(mocks.GenericEventType(0)),
			BlockIds: ids,
		}
		resp, err := s.GetEventsForBlockIDs(context.Background(), req)

		assert.NoError(t, err)

		assert.Len(t, resp.Results, len(blockIDs))
		for _, block := range resp.Results {
			assert.Len(t, block.Events, len(events))
		}
	})

	t.Run("nominal case without event type", func(t *testing.T) {
		t.Parallel()

		events := mocks.GenericEvents(6)
		blockIDs := mocks.GenericIdentifiers(4)
		blocks := map[flow.Identifier]uint64{
			blockIDs[0]: mocks.GenericHeight,
			blockIDs[1]: mocks.GenericHeight + 1,
			blockIDs[2]: mocks.GenericHeight + 2,
			blockIDs[3]: mocks.GenericHeight + 3,
		}

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Contains(t, blocks, blockID)

			return blocks[blockID], nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.InDelta(t, mocks.GenericHeight, height, float64(len(blocks)))
			assert.Empty(t, types)

			return events, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.InDelta(t, mocks.GenericHeight, height, float64(len(blocks)))

			return mocks.GenericHeader, nil
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

		assert.NoError(t, err)

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
		for _, id := range mocks.GenericIdentifiers(4) {
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
		for _, id := range mocks.GenericIdentifiers(4) {
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
		for _, id := range mocks.GenericIdentifiers(4) {
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
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		events := mocks.GenericEvents(6)

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(h uint64, types ...flow.EventType) ([]flow.Event, error) {
			// Expect height to be between GenericHeight and GenericHeight + 3 since there are four
			// given blockIDs.
			assert.InDelta(t, mocks.GenericHeight, h, 3)
			assert.Equal(t, mocks.GenericEventTypes(1), types)

			return events, nil
		}
		index.HeaderFunc = func(h uint64) (*flow.Header, error) {
			// Expect height to be between GenericHeight and GenericHeight + 3 since there are four
			// given blockIDs.
			assert.InDelta(t, mocks.GenericHeight, h, 3)

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: mocks.GenericHeight,
			EndHeight:   mocks.GenericHeight + 3,
			Type:        string(mocks.GenericEventType(0)),
		}
		resp, err := s.GetEventsForHeightRange(context.Background(), req)

		assert.NoError(t, err)

		assert.Len(t, resp.Results, 4)
		for _, block := range resp.Results {
			assert.Len(t, block.Events, len(events))
		}
	})

	t.Run("nominal case without event type", func(t *testing.T) {
		t.Parallel()

		events := mocks.GenericEvents(6)

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(h uint64, types ...flow.EventType) ([]flow.Event, error) {
			// Expect height to be between GenericHeight and GenericHeight + 3 since there are four
			// given blockIDs.
			assert.InDelta(t, mocks.GenericHeight, h, 3)
			assert.Empty(t, types)

			return events, nil
		}
		index.HeaderFunc = func(h uint64) (*flow.Header, error) {
			// Expect height to be between GenericHeight and GenericHeight + 3 since there are four
			// given blockIDs.
			assert.InDelta(t, mocks.GenericHeight, h, 3)

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: mocks.GenericHeight,
			EndHeight:   mocks.GenericHeight + 3,
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
			StartHeight: mocks.GenericHeight,
			EndHeight:   mocks.GenericHeight + 3,
			Type:        string(mocks.GenericEventType(0)),
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
			StartHeight: mocks.GenericHeight,
			EndHeight:   mocks.GenericHeight + 3,
			Type:        string(mocks.GenericEventType(0)),
		}
		_, err := s.GetEventsForHeightRange(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetNetworkParameters(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, height, mocks.GenericHeight)

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetNetworkParametersRequest{}
		resp, err := s.GetNetworkParameters(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, dps.FlowTestnet.String(), resp.ChainId)
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
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.CollectionFunc = func(collID flow.Identifier) (*flow.LightCollection, error) {
			assert.Equal(t, mocks.GenericIdentifier(0), collID)

			return mocks.GenericCollection(0), nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetCollectionByIDRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
		resp, err := s.GetCollectionByID(context.Background(), req)

		assert.NoError(t, err)

		assert.NotNil(t, resp.Collection)
		for i, txID := range resp.Collection.TransactionIds {
			assert.Equal(t, mocks.ByteSlice(mocks.GenericIdentifier(i)), txID)
		}
	})

	t.Run("handles indexer failure on collection", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.CollectionFunc = func(collID flow.Identifier) (*flow.LightCollection, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetCollectionByIDRequest{Id: mocks.ByteSlice(mocks.GenericIdentifier(0))}
		_, err := s.GetCollectionByID(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetAccount(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericLedgerValues(4), nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.GetAccountFunc = func(address flow.Address, header *flow.Header) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)
			assert.Equal(t, mocks.GenericHeader, header)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.invoker = invoker

		req := &access.GetAccountRequest{Address: mocks.GenericAccount.Address[:]}
		resp, err := s.GetAccount(context.Background(), req)

		assert.NoError(t, err)

		assert.NotNil(t, resp.Account)
		assert.Equal(t, mocks.GenericAccount.Address[:], resp.Account.Address)
		assert.Equal(t, mocks.GenericAccount.Balance, resp.Account.Balance)
	})

	t.Run("handles indexer failure on Last", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetAccountRequest{Address: mocks.GenericAccount.Address[:]}
		_, err := s.GetAccount(context.Background(), req)

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

		req := &access.GetAccountRequest{Address: mocks.GenericAccount.Address[:]}
		_, err := s.GetAccount(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles invoker failure on GetAccount", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.GetAccountFunc = func(flow.Address, *flow.Header) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.GetAccountRequest{Address: mocks.GenericAccount.Address[:]}
		_, err := s.GetAccount(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetAccountAtLatestBlock(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericLedgerValues(4), nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.GetAccountFunc = func(address flow.Address, header *flow.Header) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)
			assert.Equal(t, mocks.GenericHeader, header)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.invoker = invoker

		req := &access.GetAccountAtLatestBlockRequest{Address: mocks.GenericAccount.Address[:]}
		resp, err := s.GetAccountAtLatestBlock(context.Background(), req)

		assert.NoError(t, err)

		assert.NotNil(t, resp.Account)
		assert.Equal(t, mocks.GenericAccount.Address[:], resp.Account.Address)
		assert.Equal(t, mocks.GenericAccount.Balance, resp.Account.Balance)
	})

	t.Run("handles indexer failure on Last", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetAccountAtLatestBlockRequest{Address: mocks.GenericAccount.Address[:]}
		_, err := s.GetAccountAtLatestBlock(context.Background(), req)

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

		req := &access.GetAccountAtLatestBlockRequest{Address: mocks.GenericAccount.Address[:]}
		_, err := s.GetAccountAtLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles invoker failure on GetAccount", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.GetAccountFunc = func(flow.Address, *flow.Header) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.GetAccountAtLatestBlockRequest{Address: mocks.GenericAccount.Address[:]}
		_, err := s.GetAccountAtLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetAccountAtBlockHeight(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			assert.Equal(t, mocks.GenericHeight+999, height)

			return mocks.GenericLedgerValues(4), nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight+999, height)

			return mocks.GenericHeader, nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.GetAccountFunc = func(address flow.Address, header *flow.Header) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)
			assert.Equal(t, mocks.GenericHeader, header)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.invoker = invoker

		req := &access.GetAccountAtBlockHeightRequest{
			BlockHeight: mocks.GenericHeight + 999,
			Address:     mocks.GenericAccount.Address[:],
		}
		resp, err := s.GetAccountAtBlockHeight(context.Background(), req)

		assert.NoError(t, err)

		assert.NotNil(t, resp.Account)
		assert.Equal(t, mocks.GenericAccount.Address[:], resp.Account.Address)
		assert.Equal(t, mocks.GenericAccount.Balance, resp.Account.Balance)
	})

	t.Run("handles indexer failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetAccountAtBlockHeightRequest{
			BlockHeight: mocks.GenericHeight,
			Address:     mocks.GenericAccount.Address[:],
		}
		_, err := s.GetAccountAtBlockHeight(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles invoker failure on GetAccount", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.GetAccountFunc = func(flow.Address, *flow.Header) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.invoker = invoker

		req := &access.GetAccountAtBlockHeightRequest{
			BlockHeight: mocks.GenericHeight,
			Address:     mocks.GenericAccount.Address[:],
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

		assert.NoError(t, err)

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
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Equal(t, mocks.GenericIdentifier(0), blockID)

			return mocks.GenericHeight, nil
		}

		s := baselineServer(t)
		s.index = index
		s.invoker = invoker

		req := &access.ExecuteScriptAtBlockIDRequest{
			BlockId:   mocks.ByteSlice(mocks.GenericIdentifier(0)),
			Script:    mocks.GenericBytes,
			Arguments: [][]byte{cadenceValueBytes},
		}
		resp, err := s.ExecuteScriptAtBlockID(context.Background(), req)

		assert.NoError(t, err)

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
			BlockId:   mocks.ByteSlice(mocks.GenericIdentifier(0)),
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
			BlockId:   mocks.ByteSlice(mocks.GenericIdentifier(0)),
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

		assert.NoError(t, err)

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
	blockID := mocks.ByteSlice(mocks.GenericHeader.ID())

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var sealCalled, collCalled int
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(id flow.Identifier) (uint64, error) {
			assert.Equal(t, mocks.GenericHeader.ID(), id)

			return mocks.GenericHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}
		index.SealsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericIdentifiers(6), nil
		}
		index.SealFunc = func(sealID flow.Identifier) (*flow.Seal, error) {
			assert.Equal(t, mocks.GenericIdentifier(sealCalled), sealID)

			seal := mocks.GenericSeal(sealCalled)
			sealCalled++

			return seal, nil
		}
		index.CollectionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericIdentifiers(6), nil
		}
		index.CollectionFunc = func(collID flow.Identifier) (*flow.LightCollection, error) {
			assert.Equal(t, mocks.GenericIdentifier(collCalled), collID)

			coll := mocks.GenericCollection(collCalled)
			collCalled++

			return coll, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetLatestBlockRequest{}
		resp, err := s.GetLatestBlock(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, blockID, resp.Block.Id)
		assert.Equal(t, mocks.GenericHeight, resp.Block.Height)
		assert.Equal(t, mocks.GenericHeader.ParentID[:], resp.Block.ParentId)
		assert.NotZero(t, resp.Block.Timestamp)

		for i, id := range mocks.GenericIdentifiers(6) {
			seal := mocks.GenericSeal(i)
			wantSeal := &entities.BlockSeal{
				BlockId:                    seal.BlockID[:],
				ExecutionReceiptId:         seal.ResultID[:],
				ExecutionReceiptSignatures: [][]byte{},
			}
			assert.Contains(t, resp.Block.BlockSeals, wantSeal)

			guarantee := mocks.GenericGuarantee(i)
			wantGuarantee := &entities.CollectionGuarantee{
				CollectionId: id[:],
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
	wantBlockID := mocks.ByteSlice(mocks.GenericHeader.ID())

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var sealCalled, collCalled int
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Equal(t, wantBlockID, blockID[:])

			return mocks.GenericHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}
		index.SealsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericIdentifiers(6), nil
		}
		index.SealFunc = func(sealID flow.Identifier) (*flow.Seal, error) {
			assert.Equal(t, mocks.GenericIdentifier(sealCalled), sealID)

			seal := mocks.GenericSeal(sealCalled)
			sealCalled++

			return seal, nil
		}
		index.CollectionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericIdentifiers(6), nil
		}
		index.CollectionFunc = func(collID flow.Identifier) (*flow.LightCollection, error) {
			assert.Equal(t, mocks.GenericIdentifier(collCalled), collID)

			coll := mocks.GenericCollection(collCalled)
			collCalled++

			return coll, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: wantBlockID}
		resp, err := s.GetBlockByID(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, wantBlockID, resp.Block.Id)
		assert.Equal(t, mocks.GenericHeight, resp.Block.Height)
		assert.Equal(t, mocks.GenericHeader.ParentID[:], resp.Block.ParentId)
		assert.NotZero(t, resp.Block.Timestamp)

		for i, id := range mocks.GenericIdentifiers(6) {
			seal := mocks.GenericSeal(i)
			wantSeal := &entities.BlockSeal{
				BlockId:                    seal.BlockID[:],
				ExecutionReceiptId:         seal.ResultID[:],
				ExecutionReceiptSignatures: [][]byte{},
			}
			assert.Contains(t, resp.Block.BlockSeals, wantSeal)

			guarantee := mocks.GenericGuarantee(i)
			wantGuarantee := &entities.CollectionGuarantee{
				CollectionId: id[:],
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

		req := &access.GetBlockByIDRequest{Id: wantBlockID}
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

		req := &access.GetBlockByIDRequest{Id: wantBlockID}
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

		req := &access.GetBlockByIDRequest{Id: wantBlockID}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on Seal", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.SealFunc = func(sealID flow.Identifier) (*flow.Seal, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: wantBlockID}
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

		req := &access.GetBlockByIDRequest{Id: wantBlockID}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on Guarantee", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.GuaranteeFunc = func(colID flow.Identifier) (*flow.CollectionGuarantee, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByIDRequest{Id: wantBlockID}
		_, err := s.GetBlockByID(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetBlockByHeight(t *testing.T) {
	wantBlockID := mocks.GenericHeader.ID()

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var sealCalled, collCalled int
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Equal(t, wantBlockID, blockID[:])

			return mocks.GenericHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}
		index.SealsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericIdentifiers(6), nil
		}
		index.SealFunc = func(sealID flow.Identifier) (*flow.Seal, error) {
			assert.Equal(t, mocks.GenericIdentifier(sealCalled), sealID)

			seal := mocks.GenericSeal(sealCalled)
			sealCalled++

			return seal, nil
		}
		index.CollectionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericIdentifiers(6), nil
		}
		index.CollectionFunc = func(collID flow.Identifier) (*flow.LightCollection, error) {
			assert.Equal(t, mocks.GenericIdentifier(collCalled), collID)

			coll := mocks.GenericCollection(collCalled)
			collCalled++

			return coll, nil
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockByHeightRequest{Height: mocks.GenericHeight}
		resp, err := s.GetBlockByHeight(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, wantBlockID[:], resp.Block.Id)
		assert.Equal(t, mocks.GenericHeight, resp.Block.Height)
		assert.Equal(t, mocks.GenericHeader.ParentID[:], resp.Block.ParentId)
		assert.NotZero(t, resp.Block.Timestamp)

		for i, id := range mocks.GenericIdentifiers(6) {
			seal := mocks.GenericSeal(i)
			wantSeal := &entities.BlockSeal{
				BlockId:                    seal.BlockID[:],
				ExecutionReceiptId:         seal.ResultID[:],
				ExecutionReceiptSignatures: [][]byte{},
			}
			assert.Contains(t, resp.Block.BlockSeals, wantSeal)

			guarantee := mocks.GenericGuarantee(i)
			wantGuarantee := &entities.CollectionGuarantee{
				CollectionId: id[:],
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

		req := &access.GetBlockByHeightRequest{Height: mocks.GenericHeight}
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

		req := &access.GetBlockByHeightRequest{Height: mocks.GenericHeight}
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

		req := &access.GetBlockByHeightRequest{Height: mocks.GenericHeight}
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
