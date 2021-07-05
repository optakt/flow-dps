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

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"

	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewServer(t *testing.T) {
	index := &mocks.Reader{}
	codec := &mocks.Codec{}

	s := NewServer(index, codec)

	assert.NotNil(t, s)
	assert.NotNil(t, s.codec)
	assert.Equal(t, index, s.index)
	assert.Equal(t, codec, s.codec)
}

func TestServer_GetLatestBlockHeader(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		index := &mocks.Reader{
			LastFunc: func() (uint64, error) {
				return mocks.GenericHeight, nil
			},
			HeaderFunc: func(height uint64) (*flow.Header, error) {
				assert.Equal(t, mocks.GenericHeight, height)

				return mocks.GenericHeader, nil
			},
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
		index := &mocks.Reader{
			LastFunc: func() (uint64, error) {
				return 0, mocks.DummyError
			},
		}

		s := baselineServer(t)
		s.index = index

		_, err := s.GetLatestBlockHeader(context.Background(), nil)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on Header", func(t *testing.T) {
		index := &mocks.Reader{
			LastFunc: func() (uint64, error) {
				return mocks.GenericHeight, nil
			},
			HeaderFunc: func(height uint64) (*flow.Header, error) {
				return nil, mocks.DummyError
			},
		}

		s := baselineServer(t)
		s.index = index

		_, err := s.GetLatestBlockHeader(context.Background(), nil)

		assert.Error(t, err)
	})
}

func TestServer_GetBlockHeaderByID(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		index := &mocks.Reader{
			HeightForBlockFunc: func(blockID flow.Identifier) (uint64, error) {
				assert.Equal(t, mocks.GenericIdentifiers[0], blockID)

				return mocks.GenericHeight, nil
			},
			HeaderFunc: func(height uint64) (*flow.Header, error) {
				assert.Equal(t, mocks.GenericHeight, height)

				return mocks.GenericHeader, nil
			},
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockHeaderByIDRequest{Id: mocks.GenericIdentifiers[0][:]}
		resp, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.NoError(t, err)

		var gotID flow.Identifier
		copy(gotID[:], resp.Block.Id)
		assert.Equal(t, mocks.GenericHeader.ID(), gotID)
		assert.Equal(t, mocks.GenericHeader.Height, resp.Block.Height)
	})

	t.Run("handles indexer error on HeightForBlock", func(t *testing.T) {
		index := &mocks.Reader{
			HeightForBlockFunc: func(blockID flow.Identifier) (uint64, error) {
				return 0, mocks.DummyError
			},
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockHeaderByIDRequest{Id: mocks.GenericIdentifiers[0][:]}
		_, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer error on header", func(t *testing.T) {
		index := &mocks.Reader{
			HeightForBlockFunc: func(blockID flow.Identifier) (uint64, error) {
				return mocks.GenericHeight, nil
			},
			HeaderFunc: func(height uint64) (*flow.Header, error) {
				return nil, mocks.DummyError
			},
		}

		s := baselineServer(t)
		s.index = index

		req := &access.GetBlockHeaderByIDRequest{Id: mocks.GenericIdentifiers[0][:]}
		_, err := s.GetBlockHeaderByID(context.Background(), req)

		assert.Error(t, err)
	})
}

func TestServer_GetBlockHeaderByHeight(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		index := &mocks.Reader{
			HeaderFunc: func(height uint64) (*flow.Header, error) {
				assert.Equal(t, mocks.GenericHeight, height)

				return mocks.GenericHeader, nil
			},
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
		index := &mocks.Reader{
			HeaderFunc: func(height uint64) (*flow.Header, error) {
				return nil, mocks.DummyError
			},
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

	index := &mocks.Reader{
		FirstFunc: func() (uint64, error) {
			return mocks.GenericHeight, nil
		},
		LastFunc: func() (uint64, error) {
			return mocks.GenericHeight, nil
		},
		HeightForBlockFunc: func(blockID flow.Identifier) (uint64, error) {
			return mocks.GenericHeight, nil
		},
		CommitFunc: func(height uint64) (flow.StateCommitment, error) {
			return mocks.GenericCommits[0], nil
		},
		HeaderFunc: func(height uint64) (*flow.Header, error) {
			return mocks.GenericHeader, nil
		},
		EventsFunc: func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return mocks.GenericEvents, nil
		},
		ValuesFunc: func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			return mocks.GenericLedgerValues, nil
		},
		TransactionFunc: func(txID flow.Identifier) (*flow.TransactionBody, error) {
			return mocks.GenericTransactions[0], nil
		},
		TransactionsByHeightFunc: func(height uint64) ([]flow.Identifier, error) {
			return mocks.GenericIdentifiers, nil
		},
	}

	codec := &mocks.Codec{
		UnmarshalFunc: func(b []byte, v interface{}) error {
			return nil
		},
		MarshalFunc: func(v interface{}) ([]byte, error) {
			return []byte(`test`), nil
		},
	}

	s := Server{
		index: index,
		codec: codec,
	}

	return &s
}
