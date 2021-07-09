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

package dps

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"

	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewServer(t *testing.T) {
	index := mocks.BaselineReader(t)
	codec := mocks.BaselineCodec(t)

	s := NewServer(index, codec)

	assert.NotNil(t, s)
	assert.NotNil(t, s.codec)
	assert.Equal(t, index, s.index)
	assert.Equal(t, codec, s.codec)
}

func TestServer_GetLatestBlockHeader(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		s := baselineServer(t)
		s.index = index

		resp, err := s.GetLatestBlockHeader(context.Background(), nil)

		assert.NoError(t, err)

		var gotID flow.Identifier
		copy(gotID[:], resp.Block.Id)
		assert.Equal(t, mocks.GenericHeader.ID(), gotID)
		assert.Equal(t, mocks.GenericHeader.Height, resp.Block.Height)
	})

	t.Run("handles indexer error on Last", func(t *testing.T) {
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

		id := mocks.GenericIdentifier(0)
		req := &access.GetBlockHeaderByIDRequest{Id: id[:]}
		resp, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.NoError(t, err)

		var gotID flow.Identifier
		copy(gotID[:], resp.Block.Id)
		assert.Equal(t, mocks.GenericHeader.ID(), gotID)
		assert.Equal(t, mocks.GenericHeader.Height, resp.Block.Height)
	})

	t.Run("handles indexer error on HeightForBlock", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			return 0, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		id := mocks.GenericIdentifier(0)
		req := &access.GetBlockHeaderByIDRequest{Id: id[:]}
		_, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on header", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
			return mocks.GenericHeight, nil
		}
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		s := baselineServer(t)
		s.index = index

		id := mocks.GenericIdentifier(0)
		req := &access.GetBlockHeaderByIDRequest{Id: id[:]}
		_, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetBlockHeaderByHeight(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
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

		var gotID flow.Identifier
		copy(gotID[:], resp.Block.Id)
		assert.Equal(t, mocks.GenericHeader.ID(), gotID)
		assert.Equal(t, mocks.GenericHeader.Height, resp.Block.Height)
	})

	t.Run("handles indexer error on header", func(t *testing.T) {
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

func baselineServer(t *testing.T) *Server {
	t.Helper()

	s := Server{
		index: mocks.BaselineReader(t),
		codec: mocks.BaselineCodec(t),
	}

	return &s
}
