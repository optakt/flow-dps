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

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"
	"github.com/onflow/flow-go/fvm/state"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewServer(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		codec := mocks.BaselineCodec(t)

		s, err := NewServer(index, codec, WithChainID(dps.FlowMainnet.String()), WithCacheSize(uint64(datasize.MB)))

		assert.NoError(t, err)
		assert.NotNil(t, s)
		assert.NotNil(t, s.codec)
		assert.Equal(t, index, s.index)
		assert.Equal(t, codec, s.codec)
		assert.Equal(t, dps.FlowMainnet.String(), s.chainID)
	})

	t.Run("handles invalid cache size", func(t *testing.T) {
		t.Run("nominal case", func(t *testing.T) {
			index := mocks.BaselineReader(t)
			codec := mocks.BaselineCodec(t)

			_, err := NewServer(index, codec, WithChainID(dps.FlowMainnet.String()), WithCacheSize(0))

			assert.Error(t, err)
		})
	})
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
		index := mocks.BaselineReader(t)
		index.CollectionFunc = func(cID flow.Identifier) (*flow.LightCollection, error) {
			assert.Equal(t, mocks.GenericIdentifier(0), cID)

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
		index := mocks.BaselineReader(t)
		index.CollectionFunc = func(cID flow.Identifier) (*flow.LightCollection, error) {
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
	t.Run("nominal case account not cached", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericLedgerValues(4), nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			assert.Equal(t, mocks.GenericBytes, key)

			return mocks.GenericBytes, false
		}
		cache.SetFunc = func(key, value interface{}, cost int64) bool {
			assert.Equal(t, mocks.GenericBytes, key)

			return true
		}

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(ctx fvm.Context, address flow.Address, v state.View, programs *programs.Programs) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.cache = cache
		s.vm = vm

		req := &access.GetAccountRequest{Address: mocks.GenericAccount.Address[:]}
		resp, err := s.GetAccount(context.Background(), req)

		assert.NoError(t, err)

		assert.NotNil(t, resp.Account)
		assert.Equal(t, mocks.GenericAccount.Address[:], resp.Account.Address)
		assert.Equal(t, mocks.GenericAccount.Balance, resp.Account.Balance)
	})

	t.Run("nominal case account cached", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(ctx fvm.Context, address flow.Address, v state.View, programs *programs.Programs) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.vm = vm

		req := &access.GetAccountRequest{Address: mocks.GenericAccount.Address[:]}
		resp, err := s.GetAccount(context.Background(), req)

		assert.NoError(t, err)

		assert.NotNil(t, resp.Account)
		assert.Equal(t, mocks.GenericAccount.Address[:], resp.Account.Address)
		assert.Equal(t, mocks.GenericAccount.Balance, resp.Account.Balance)
	})

	t.Run("handles indexer failure on Last", func(t *testing.T) {
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

	t.Run("handles vm failure on GetAccount", func(t *testing.T) {
		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(fvm.Context, flow.Address, state.View, *programs.Programs) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.vm = vm

		req := &access.GetAccountRequest{Address: mocks.GenericAccount.Address[:]}
		_, err := s.GetAccount(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetAccountAtLatestBlock(t *testing.T) {
	t.Run("nominal case account not cached", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericLedgerValues(4), nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			assert.Equal(t, mocks.GenericBytes, key)

			return mocks.GenericBytes, false
		}
		cache.SetFunc = func(key, value interface{}, cost int64) bool {
			assert.Equal(t, mocks.GenericBytes, key)

			return true
		}

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(ctx fvm.Context, address flow.Address, v state.View, programs *programs.Programs) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.cache = cache
		s.vm = vm

		req := &access.GetAccountAtLatestBlockRequest{Address: mocks.GenericAccount.Address[:]}
		resp, err := s.GetAccountAtLatestBlock(context.Background(), req)

		assert.NoError(t, err)

		assert.NotNil(t, resp.Account)
		assert.Equal(t, mocks.GenericAccount.Address[:], resp.Account.Address)
		assert.Equal(t, mocks.GenericAccount.Balance, resp.Account.Balance)
	})

	t.Run("nominal case account cached", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(ctx fvm.Context, address flow.Address, v state.View, programs *programs.Programs) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.vm = vm

		req := &access.GetAccountAtLatestBlockRequest{Address: mocks.GenericAccount.Address[:]}
		resp, err := s.GetAccountAtLatestBlock(context.Background(), req)

		assert.NoError(t, err)

		assert.NotNil(t, resp.Account)
		assert.Equal(t, mocks.GenericAccount.Address[:], resp.Account.Address)
		assert.Equal(t, mocks.GenericAccount.Balance, resp.Account.Balance)
	})

	t.Run("handles indexer failure on Last", func(t *testing.T) {
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

	t.Run("handles vm failure on GetAccount", func(t *testing.T) {
		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(fvm.Context, flow.Address, state.View, *programs.Programs) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.vm = vm

		req := &access.GetAccountAtLatestBlockRequest{Address: mocks.GenericAccount.Address[:]}
		_, err := s.GetAccountAtLatestBlock(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetAccountAtBlockHeight(t *testing.T) {
	t.Run("nominal case account not cached", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			assert.Equal(t, mocks.GenericHeight+999, height)

			return mocks.GenericLedgerValues(4), nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight+999, height)

			return mocks.GenericHeader, nil
		}

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			assert.Equal(t, mocks.GenericBytes, key)

			return mocks.GenericBytes, false
		}
		cache.SetFunc = func(key, value interface{}, cost int64) bool {
			assert.Equal(t, mocks.GenericBytes, key)

			return true
		}

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(ctx fvm.Context, address flow.Address, v state.View, programs *programs.Programs) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.cache = cache
		s.vm = vm

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

	t.Run("nominal case account cached", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight+999, height)

			return mocks.GenericHeader, nil
		}

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(ctx fvm.Context, address flow.Address, v state.View, programs *programs.Programs) (*flow.Account, error) {
			assert.Equal(t, mocks.GenericAccount.Address, address)

			return &mocks.GenericAccount, nil
		}

		s := baselineServer(t)
		s.index = index
		s.vm = vm

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

	t.Run("handles vm failure on GetAccount", func(t *testing.T) {
		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(fvm.Context, flow.Address, state.View, *programs.Programs) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.vm = vm

		req := &access.GetAccountAtBlockHeightRequest{
			BlockHeight: mocks.GenericHeight,
			Address:     mocks.GenericAccount.Address[:],
		}
		_, err := s.GetAccountAtBlockHeight(context.Background(), req)

		assert.Error(t, err)
	})
}

func baselineServer(t *testing.T) *Server {
	t.Helper()

	s := Server{
		cache:   mocks.BaselineCache(t),
		chainID: dps.FlowMainnet.String(),
		codec:   mocks.BaselineCodec(t),
		index:   mocks.BaselineReader(t),
		vm:      mocks.BaselineVirtualMachine(t),
	}

	return &s
}
