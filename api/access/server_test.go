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

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewServer(t *testing.T) {
	index := mocks.BaselineReader(t)
	codec := mocks.BaselineCodec(t)

	s := NewServer(index, codec, dps.FlowMainnet.String())

	assert.NotNil(t, s)
	assert.NotNil(t, s.codec)
	assert.Equal(t, index, s.index)
	assert.Equal(t, codec, s.codec)
	assert.Equal(t, dps.FlowMainnet.String(), s.chainID)
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

func TestServer_GetEventsForBlockIDs(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		blockIDs := mocks.GenericIdentifiers(4)
		events := mocks.GenericEvents(6)
		height := mocks.GenericHeight

		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			assert.Contains(t, blockIDs, blockID)
			height++

			return height, nil
		}
		index.EventsFunc = func(h uint64, types ...flow.EventType) ([]flow.Event, error) {
			// Expect height to be between GenericHeight and GenericHeight + 4 since there are four
			// given blockIDs.
			assert.InDelta(t, mocks.GenericHeight, h, 4)
			assert.Empty(t, types)

			return events, nil
		}
		index.HeaderFunc = func(h uint64) (*flow.Header, error) {
			// Expect height to be between GenericHeight and GenericHeight + 4 since there are four
			// given blockIDs.
			assert.InDelta(t, mocks.GenericHeight, h, 4)

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		var ids [][]byte
		for _, id := range blockIDs {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForBlockIDsRequest{BlockIds: ids}
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
		req := &access.GetEventsForBlockIDsRequest{BlockIds: ids}
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
		req := &access.GetEventsForBlockIDsRequest{BlockIds: ids}
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
		req := &access.GetEventsForBlockIDsRequest{BlockIds: ids}
		_, err := s.GetEventsForBlockIDs(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetEventsForHeightRange(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		blockIDs := mocks.GenericIdentifiers(4)
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

		var ids [][]byte
		for _, id := range blockIDs {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: mocks.GenericHeight,
			EndHeight:   mocks.GenericHeight + 3,
		}
		resp, err := s.GetEventsForHeightRange(context.Background(), req)

		assert.NoError(t, err)

		assert.Len(t, resp.Results, len(blockIDs))
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

		var ids [][]byte
		for _, id := range mocks.GenericIdentifiers(4) {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: mocks.GenericHeight,
			EndHeight:   mocks.GenericHeight + 3,
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

		var ids [][]byte
		for _, id := range mocks.GenericIdentifiers(4) {
			ids = append(ids, id[:])
		}
		req := &access.GetEventsForHeightRangeRequest{
			StartHeight: mocks.GenericHeight,
			EndHeight:   mocks.GenericHeight + 3,
		}
		_, err := s.GetEventsForHeightRange(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetNetworkParameters(t *testing.T) {
	s := baselineServer(t)

	req := &access.GetNetworkParametersRequest{}
	resp, err := s.GetNetworkParameters(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, dps.FlowMainnet.String(), resp.ChainId)
}

func baselineServer(t *testing.T) *Server {
	t.Helper()

	s := Server{
		index:   mocks.BaselineReader(t),
		codec:   mocks.BaselineCodec(t),
		chainID: dps.FlowMainnet.String(),
	}

	return &s
}
