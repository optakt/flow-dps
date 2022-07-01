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

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/onflow/flow-dps/models/convert"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestIndexFromAPI(t *testing.T) {
	mock := &apiMock{}
	codec := mocks.BaselineCodec(t)

	index := IndexFromAPI(mock, codec)

	require.NotNil(t, index)
	assert.Equal(t, mock, index.client)
	assert.NotNil(t, mock, index.codec)
}

func TestIndex_First(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetFirstFunc: func(_ context.Context, in *GetFirstRequest, _ ...grpc.CallOption) (*GetFirstResponse, error) {
					assert.NotNil(t, in)

					return &GetFirstResponse{
						Height: mocks.GenericHeight,
					}, nil
				},
			},
		}

		got, err := index.First()

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("handles index failure", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetFirstFunc: func(context.Context, *GetFirstRequest, ...grpc.CallOption) (*GetFirstResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.First()
		assert.Error(t, err)
	})
}

func TestIndex_Last(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetLastFunc: func(_ context.Context, in *GetLastRequest, _ ...grpc.CallOption) (*GetLastResponse, error) {
					assert.NotNil(t, in)

					return &GetLastResponse{
						Height: mocks.GenericHeight,
					}, nil
				},
			},
		}

		got, err := index.Last()

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeight, got)
	})

	t.Run("handles index failure", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetLastFunc: func(context.Context, *GetLastRequest, ...grpc.CallOption) (*GetLastResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Last()
		assert.Error(t, err)
	})
}

func TestIndex_Header(t *testing.T) {
	// We need to use the proper encoding to support nanoseconds
	// and timezones in timestamps.
	options := cbor.CanonicalEncOptions()
	options.Time = cbor.TimeRFC3339Nano
	encoder, err := options.EncMode()
	require.NoError(t, err)

	header := mocks.GenericHeader
	data, err := encoder.Marshal(header)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetHeaderFunc: func(_ context.Context, in *GetHeaderRequest, _ ...grpc.CallOption) (*GetHeaderResponse, error) {
					assert.Equal(t, header.Height, in.Height)

					return &GetHeaderResponse{
						Height: header.Height,
						Data:   data,
					}, nil
				},
			},
		}

		got, err := index.Header(header.Height)

		require.NoError(t, err)
		assert.Equal(t, header, got)
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				GetHeaderFunc: func(context.Context, *GetHeaderRequest, ...grpc.CallOption) (*GetHeaderResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Header(header.Height)

		assert.Error(t, err)
	})

	t.Run("handles decoding failures", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetHeaderFunc: func(_ context.Context, in *GetHeaderRequest, _ ...grpc.CallOption) (*GetHeaderResponse, error) {
					assert.Equal(t, header.Height, in.Height)

					return &GetHeaderResponse{
						Height: header.Height,
						Data:   []byte(`invalid data`),
					}, nil
				},
			},
		}

		_, err := index.Header(header.Height)

		assert.Error(t, err)
	})
}

func TestIndex_Commit(t *testing.T) {
	commit := mocks.GenericCommit(0)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetCommitFunc: func(_ context.Context, in *GetCommitRequest, _ ...grpc.CallOption) (*GetCommitResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)

					return &GetCommitResponse{
						Height: mocks.GenericHeight,
						Commit: commit[:],
					}, nil
				},
			},
		}

		got, err := index.Commit(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Equal(t, commit, got)
	})

	codec := mocks.BaselineCodec(t)
	codec.UnmarshalFunc = cbor.Unmarshal

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetCommitFunc: func(context.Context, *GetCommitRequest, ...grpc.CallOption) (*GetCommitResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Commit(mocks.GenericHeight)

		assert.Error(t, err)
	})

	t.Run("handles invalid indexed data", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			client: &apiMock{
				GetCommitFunc: func(_ context.Context, in *GetCommitRequest, _ ...grpc.CallOption) (*GetCommitResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)

					return &GetCommitResponse{
						Height: mocks.GenericHeight,
						Commit: []byte(`not a commit`),
					}, nil
				},
			},
		}

		_, err := index.Commit(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestIndex_Values(t *testing.T) {
	paths := mocks.GenericLedgerPaths(6)
	values := mocks.GenericLedgerValues(6)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetRegisterValuesFunc: func(_ context.Context, in *GetRegisterValuesRequest, _ ...grpc.CallOption) (*GetRegisterValuesResponse, error) {
					require.Len(t, in.Paths, 6)
					codec := mocks.BaselineCodec(t)
					codec.UnmarshalFunc = cbor.Unmarshal

					assert.Equal(t, convert.PathsToBytes(paths), in.Paths)
					assert.Equal(t, in.Height, mocks.GenericHeight)

					return &GetRegisterValuesResponse{
						Height: mocks.GenericHeight,
						Paths:  convert.PathsToBytes(paths),
						Values: convert.ValuesToBytes(values),
					}, nil
				},
			},
		}

		got, err := index.Values(mocks.GenericHeight, paths)

		require.NoError(t, err)
		assert.Equal(t, values, got)
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal
		index := Index{
			client: &apiMock{
				GetRegisterValuesFunc: func(context.Context, *GetRegisterValuesRequest, ...grpc.CallOption) (*GetRegisterValuesResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Values(mocks.GenericHeight, paths)

		assert.Error(t, err)
	})
}

func TestIndex_Height(t *testing.T) {
	header := mocks.GenericHeader
	blockID := header.ID()

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			client: &apiMock{
				GetHeightForBlockFunc: func(_ context.Context, in *GetHeightForBlockRequest, _ ...grpc.CallOption) (*GetHeightForBlockResponse, error) {
					assert.Equal(t, blockID[:], in.BlockID)

					return &GetHeightForBlockResponse{
						BlockID: blockID[:],
						Height:  header.Height,
					}, nil
				},
			},
		}

		got, err := index.HeightForBlock(blockID)

		require.NoError(t, err)
		assert.Equal(t, header.Height, got)
	})

	codec := mocks.BaselineCodec(t)
	codec.UnmarshalFunc = cbor.Unmarshal

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetHeightForBlockFunc: func(context.Context, *GetHeightForBlockRequest, ...grpc.CallOption) (*GetHeightForBlockResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.HeightForBlock(blockID)

		assert.Error(t, err)
	})
}

func TestIndex_Collection(t *testing.T) {
	collection := mocks.GenericCollection(0)
	collID := collection.ID()
	data, err := cbor.Marshal(collection)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetCollectionFunc: func(_ context.Context, in *GetCollectionRequest, _ ...grpc.CallOption) (*GetCollectionResponse, error) {
					assert.Equal(t, collID[:], in.CollectionID)

					return &GetCollectionResponse{
						CollectionID: collID[:],
						Data:         data,
					}, nil
				},
			},
		}

		got, err := index.Collection(collID)

		require.NoError(t, err)
		assert.Equal(t, collection, got)
	})

	t.Run("handles index failure on GetCollection", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetCollectionFunc: func(context.Context, *GetCollectionRequest, ...grpc.CallOption) (*GetCollectionResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Collection(collID)

		assert.Error(t, err)
	})
}

func TestIndex_ListCollectionsForHeight(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		collIDs := mocks.GenericCollectionIDs(4)
		data := make([][]byte, 0, len(collIDs))
		for _, collID := range collIDs {
			collID := collID
			data = append(data, collID[:])
		}

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				ListCollectionsForHeightFunc: func(_ context.Context, in *ListCollectionsForHeightRequest, _ ...grpc.CallOption) (*ListCollectionsForHeightResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)

					return &ListCollectionsForHeightResponse{
						Height:        in.Height,
						CollectionIDs: data,
					}, nil
				},
			},
		}

		got, err := index.CollectionsByHeight(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Equal(t, collIDs, got)
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				ListCollectionsForHeightFunc: func(context.Context, *ListCollectionsForHeightRequest, ...grpc.CallOption) (*ListCollectionsForHeightResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.CollectionsByHeight(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestIndex_Guarantee(t *testing.T) {
	guarantee := mocks.GenericGuarantee(0)
	collID := guarantee.ID()
	testGuaranteeBytes, err := cbor.Marshal(guarantee)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetGuaranteeFunc: func(_ context.Context, in *GetGuaranteeRequest, _ ...grpc.CallOption) (*GetGuaranteeResponse, error) {
					assert.Equal(t, collID[:], in.CollectionID)

					return &GetGuaranteeResponse{
						CollectionID: collID[:],
						Data:         testGuaranteeBytes,
					}, nil
				},
			},
		}

		got, err := index.Guarantee(collID)

		require.NoError(t, err)
		assert.Equal(t, guarantee, got)
	})

	t.Run("handles index failure on GetGuarantee", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetGuaranteeFunc: func(context.Context, *GetGuaranteeRequest, ...grpc.CallOption) (*GetGuaranteeResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Guarantee(guarantee.ID())

		assert.Error(t, err)
	})
}

func TestIndex_Transaction(t *testing.T) {
	tx := mocks.GenericTransaction(0)
	txID := tx.ID()

	data, err := cbor.Marshal(tx)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetTransactionFunc: func(_ context.Context, in *GetTransactionRequest, _ ...grpc.CallOption) (*GetTransactionResponse, error) {
					assert.Equal(t, txID[:], in.TransactionID)

					return &GetTransactionResponse{
						TransactionID: txID[:],
						Data:          data,
					}, nil
				},
			},
		}

		got, err := index.Transaction(txID)

		require.NoError(t, err)
		assert.Equal(t, tx, got)
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetTransactionFunc: func(context.Context, *GetTransactionRequest, ...grpc.CallOption) (*GetTransactionResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Transaction(tx.ID())

		assert.Error(t, err)
	})
}

func TestIndex_Result(t *testing.T) {
	result := mocks.GenericResult(0)
	txID := result.TransactionID

	data, err := cbor.Marshal(result)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetResultFunc: func(_ context.Context, in *GetResultRequest, _ ...grpc.CallOption) (*GetResultResponse, error) {
					assert.Equal(t, txID[:], in.TransactionID)

					return &GetResultResponse{
						TransactionID: txID[:],
						Data:          data,
					}, nil
				},
			},
		}

		got, err := index.Result(txID)

		require.NoError(t, err)
		assert.Equal(t, result, got)
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetResultFunc: func(context.Context, *GetResultRequest, ...grpc.CallOption) (*GetResultResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Result(result.ID())

		assert.Error(t, err)
	})
}

func TestIndex_Events(t *testing.T) {
	events := mocks.GenericEvents(4)
	types := mocks.GenericEventTypes(2)

	data, err := cbor.Marshal(events)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetEventsFunc: func(_ context.Context, in *GetEventsRequest, _ ...grpc.CallOption) (*GetEventsResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)
					assert.Equal(t, convert.TypesToStrings(types), in.Types)

					return &GetEventsResponse{
						Height: mocks.GenericHeight,
						Types:  convert.TypesToStrings(types),
						Data:   data,
					}, nil
				},
			},
		}

		got, err := index.Events(mocks.GenericHeight, types...)

		require.NoError(t, err)
		assert.Equal(t, events, got)
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetEventsFunc: func(context.Context, *GetEventsRequest, ...grpc.CallOption) (*GetEventsResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Events(mocks.GenericHeight, types...)

		assert.Error(t, err)
	})

	t.Run("handles invalid indexed data", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetEventsFunc: func(context.Context, *GetEventsRequest, ...grpc.CallOption) (*GetEventsResponse, error) {
					return &GetEventsResponse{
						Height: mocks.GenericHeight,
						Types:  convert.TypesToStrings(types),
						Data:   []byte(`invalid data`),
					}, nil
				},
			},
		}

		_, err := index.Events(mocks.GenericHeight, types...)

		assert.Error(t, err)
	})
}

func TestIndex_Seals(t *testing.T) {
	seal := mocks.GenericSeal(0)
	sealID := seal.ID()

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		data, err := cbor.Marshal(seal)
		require.NoError(t, err)

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetSealFunc: func(_ context.Context, in *GetSealRequest, _ ...grpc.CallOption) (*GetSealResponse, error) {
					assert.Equal(t, sealID[:], in.SealID)

					return &GetSealResponse{
						SealID: sealID[:],
						Data:   data,
					}, nil
				},
			},
		}

		got, err := index.Seal(sealID)

		require.NoError(t, err)
		assert.Equal(t, seal, got)
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				GetSealFunc: func(context.Context, *GetSealRequest, ...grpc.CallOption) (*GetSealResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Seal(sealID)

		assert.Error(t, err)
	})
}

func TestIndex_ListSealsForHeight(t *testing.T) {
	sealIDs := mocks.GenericSealIDs(4)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		data := make([][]byte, 0, len(sealIDs))
		for _, sealID := range sealIDs {
			data = append(data, mocks.ByteSlice(sealID))
		}

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				ListSealsForHeightFunc: func(_ context.Context, in *ListSealsForHeightRequest, _ ...grpc.CallOption) (*ListSealsForHeightResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)

					return &ListSealsForHeightResponse{
						Height:  in.Height,
						SealIDs: data,
					}, nil
				},
			},
		}

		got, err := index.SealsByHeight(mocks.GenericHeight)

		require.NoError(t, err)
		assert.Equal(t, sealIDs, got)
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				ListSealsForHeightFunc: func(context.Context, *ListSealsForHeightRequest, ...grpc.CallOption) (*ListSealsForHeightResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.SealsByHeight(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

type apiMock struct {
	GetFirstFunc                  func(ctx context.Context, in *GetFirstRequest, opts ...grpc.CallOption) (*GetFirstResponse, error)
	GetLastFunc                   func(ctx context.Context, in *GetLastRequest, opts ...grpc.CallOption) (*GetLastResponse, error)
	GetHeightForBlockFunc         func(ctx context.Context, in *GetHeightForBlockRequest, opts ...grpc.CallOption) (*GetHeightForBlockResponse, error)
	GetCommitFunc                 func(ctx context.Context, in *GetCommitRequest, opts ...grpc.CallOption) (*GetCommitResponse, error)
	GetHeaderFunc                 func(ctx context.Context, in *GetHeaderRequest, opts ...grpc.CallOption) (*GetHeaderResponse, error)
	GetEventsFunc                 func(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (*GetEventsResponse, error)
	GetRegisterValuesFunc         func(ctx context.Context, in *GetRegisterValuesRequest, opts ...grpc.CallOption) (*GetRegisterValuesResponse, error)
	GetCollectionFunc             func(ctx context.Context, in *GetCollectionRequest, opts ...grpc.CallOption) (*GetCollectionResponse, error)
	ListCollectionsForHeightFunc  func(ctx context.Context, in *ListCollectionsForHeightRequest, opts ...grpc.CallOption) (*ListCollectionsForHeightResponse, error)
	GetGuaranteeFunc              func(ctx context.Context, in *GetGuaranteeRequest, opts ...grpc.CallOption) (*GetGuaranteeResponse, error)
	GetTransactionFunc            func(ctx context.Context, in *GetTransactionRequest, opts ...grpc.CallOption) (*GetTransactionResponse, error)
	GetHeightForTransactionFunc   func(ctx context.Context, in *GetHeightForTransactionRequest, opts ...grpc.CallOption) (*GetHeightForTransactionResponse, error)
	ListTransactionsForHeightFunc func(ctx context.Context, in *ListTransactionsForHeightRequest, opts ...grpc.CallOption) (*ListTransactionsForHeightResponse, error)
	GetResultFunc                 func(ctx context.Context, in *GetResultRequest, opts ...grpc.CallOption) (*GetResultResponse, error)
	GetSealFunc                   func(ctx context.Context, in *GetSealRequest, opts ...grpc.CallOption) (*GetSealResponse, error)
	ListSealsForHeightFunc        func(ctx context.Context, in *ListSealsForHeightRequest, opts ...grpc.CallOption) (*ListSealsForHeightResponse, error)
}

func (a *apiMock) GetFirst(ctx context.Context, in *GetFirstRequest, opts ...grpc.CallOption) (*GetFirstResponse, error) {
	return a.GetFirstFunc(ctx, in, opts...)
}

func (a *apiMock) GetLast(ctx context.Context, in *GetLastRequest, opts ...grpc.CallOption) (*GetLastResponse, error) {
	return a.GetLastFunc(ctx, in, opts...)
}

func (a *apiMock) GetHeightForBlock(ctx context.Context, in *GetHeightForBlockRequest, opts ...grpc.CallOption) (*GetHeightForBlockResponse, error) {
	return a.GetHeightForBlockFunc(ctx, in, opts...)
}

func (a *apiMock) GetCommit(ctx context.Context, in *GetCommitRequest, opts ...grpc.CallOption) (*GetCommitResponse, error) {
	return a.GetCommitFunc(ctx, in, opts...)
}

func (a *apiMock) GetHeader(ctx context.Context, in *GetHeaderRequest, opts ...grpc.CallOption) (*GetHeaderResponse, error) {
	return a.GetHeaderFunc(ctx, in, opts...)
}

func (a *apiMock) GetEvents(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (*GetEventsResponse, error) {
	return a.GetEventsFunc(ctx, in, opts...)
}

func (a *apiMock) GetRegisterValues(ctx context.Context, in *GetRegisterValuesRequest, opts ...grpc.CallOption) (*GetRegisterValuesResponse, error) {
	return a.GetRegisterValuesFunc(ctx, in, opts...)
}

func (a *apiMock) GetCollection(ctx context.Context, in *GetCollectionRequest, opts ...grpc.CallOption) (*GetCollectionResponse, error) {
	return a.GetCollectionFunc(ctx, in, opts...)
}

func (a *apiMock) ListCollectionsForHeight(ctx context.Context, in *ListCollectionsForHeightRequest, opts ...grpc.CallOption) (*ListCollectionsForHeightResponse, error) {
	return a.ListCollectionsForHeightFunc(ctx, in, opts...)
}

func (a *apiMock) GetGuarantee(ctx context.Context, in *GetGuaranteeRequest, opts ...grpc.CallOption) (*GetGuaranteeResponse, error) {
	return a.GetGuaranteeFunc(ctx, in, opts...)
}

func (a *apiMock) GetTransaction(ctx context.Context, in *GetTransactionRequest, opts ...grpc.CallOption) (*GetTransactionResponse, error) {
	return a.GetTransactionFunc(ctx, in, opts...)
}

func (a *apiMock) GetHeightForTransaction(ctx context.Context, in *GetHeightForTransactionRequest, opts ...grpc.CallOption) (*GetHeightForTransactionResponse, error) {
	return a.GetHeightForTransactionFunc(ctx, in, opts...)
}

func (a *apiMock) ListTransactionsForHeight(ctx context.Context, in *ListTransactionsForHeightRequest, opts ...grpc.CallOption) (*ListTransactionsForHeightResponse, error) {
	return a.ListTransactionsForHeightFunc(ctx, in, opts...)
}

func (a *apiMock) GetResult(ctx context.Context, in *GetResultRequest, opts ...grpc.CallOption) (*GetResultResponse, error) {
	return a.GetResultFunc(ctx, in, opts...)
}

func (a *apiMock) GetSeal(ctx context.Context, in *GetSealRequest, opts ...grpc.CallOption) (*GetSealResponse, error) {
	return a.GetSealFunc(ctx, in, opts...)
}

func (a *apiMock) ListSealsForHeight(ctx context.Context, in *ListSealsForHeightRequest, opts ...grpc.CallOption) (*ListSealsForHeightResponse, error) {
	return a.ListSealsForHeightFunc(ctx, in, opts...)
}
