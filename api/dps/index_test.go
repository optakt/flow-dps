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

	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestIndexFromAPI(t *testing.T) {
	mock := &apiMock{}
	codec := mocks.BaselineCodec(t)

	index := IndexFromAPI(mock, codec)

	if assert.NotNil(t, index) {
		assert.Equal(t, mock, index.client)
		assert.NotNil(t, mock, index.codec)
	}
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

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericHeight, got)
		}
	})

	t.Run("handles index failure", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetFirstFunc: func(_ context.Context, in *GetFirstRequest, _ ...grpc.CallOption) (*GetFirstResponse, error) {
					assert.NotNil(t, in)

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

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericHeight, got)
		}
	})

	t.Run("handles index failure", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetLastFunc: func(_ context.Context, in *GetLastRequest, _ ...grpc.CallOption) (*GetLastResponse, error) {
					assert.NotNil(t, in)

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

	headerBytes, err := encoder.Marshal(mocks.GenericHeader)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetHeaderFunc: func(_ context.Context, in *GetHeaderRequest, _ ...grpc.CallOption) (*GetHeaderResponse, error) {
					assert.Equal(t, mocks.GenericHeader.Height, in.Height)

					return &GetHeaderResponse{
						Height: mocks.GenericHeader.Height,
						Data:   headerBytes,
					}, nil
				},
			},
		}

		got, err := index.Header(mocks.GenericHeader.Height)

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericHeader, got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				GetHeaderFunc: func(_ context.Context, in *GetHeaderRequest, _ ...grpc.CallOption) (*GetHeaderResponse, error) {
					assert.Equal(t, mocks.GenericHeader.Height, in.Height)

					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Header(mocks.GenericHeader.Height)

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
					assert.Equal(t, mocks.GenericHeader.Height, in.Height)

					return &GetHeaderResponse{
						Height: mocks.GenericHeader.Height,
						Data:   []byte(`invalid data`),
					}, nil
				},
			},
		}

		_, err := index.Header(mocks.GenericHeader.Height)

		assert.Error(t, err)
	})
}

func TestIndex_Commit(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetCommitFunc: func(_ context.Context, in *GetCommitRequest, _ ...grpc.CallOption) (*GetCommitResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)

					return &GetCommitResponse{
						Height: mocks.GenericHeight,
						Commit: mocks.ByteSlice(mocks.GenericCommit(0)),
					}, nil
				},
			},
		}

		got, err := index.Commit(mocks.GenericHeight)

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericCommit(0), got)
		}
	})

	codec := mocks.BaselineCodec(t)
	codec.UnmarshalFunc = cbor.Unmarshal

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetCommitFunc: func(_ context.Context, in *GetCommitRequest, _ ...grpc.CallOption) (*GetCommitResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)

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
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetRegisterValuesFunc: func(_ context.Context, in *GetRegisterValuesRequest, _ ...grpc.CallOption) (*GetRegisterValuesResponse, error) {
					if assert.Len(t, in.Paths, 6) {

						codec := mocks.BaselineCodec(t)
						codec.UnmarshalFunc = cbor.Unmarshal

						assert.Equal(t, convert.PathsToBytes(mocks.GenericLedgerPaths(6)), in.Paths)
					}
					assert.Equal(t, in.Height, mocks.GenericHeight)

					return &GetRegisterValuesResponse{
						Height: mocks.GenericHeight,
						Paths:  convert.PathsToBytes(mocks.GenericLedgerPaths(6)),
						Values: convert.ValuesToBytes(mocks.GenericLedgerValues(6)),
					}, nil
				},
			},
		}

		got, err := index.Values(mocks.GenericHeight, mocks.GenericLedgerPaths(6))

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericLedgerValues(6), got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal
		index := Index{
			client: &apiMock{
				GetRegisterValuesFunc: func(_ context.Context, in *GetRegisterValuesRequest, _ ...grpc.CallOption) (*GetRegisterValuesResponse, error) {
					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Values(mocks.GenericHeight, mocks.GenericLedgerPaths(6))

		assert.Error(t, err)
	})
}

func TestIndex_Height(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			client: &apiMock{
				GetHeightForBlockFunc: func(_ context.Context, in *GetHeightForBlockRequest, _ ...grpc.CallOption) (*GetHeightForBlockResponse, error) {
					assert.Equal(t, mocks.ByteSlice(mocks.GenericIdentifier(0)), in.BlockID)

					return &GetHeightForBlockResponse{
						BlockID: mocks.ByteSlice(mocks.GenericIdentifier(0)),
						Height:  mocks.GenericHeight,
					}, nil
				},
			},
		}

		got, err := index.HeightForBlock(mocks.GenericIdentifier(0))

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericHeight, got)
		}
	})

	codec := mocks.BaselineCodec(t)
	codec.UnmarshalFunc = cbor.Unmarshal

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetHeightForBlockFunc: func(_ context.Context, in *GetHeightForBlockRequest, _ ...grpc.CallOption) (*GetHeightForBlockResponse, error) {
					assert.Equal(t, mocks.ByteSlice(mocks.GenericIdentifier(0)), in.BlockID)

					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.HeightForBlock(mocks.GenericIdentifier(0))

		assert.Error(t, err)
	})
}

func TestIndex_Transaction(t *testing.T) {
	testTransactionBytes, err := cbor.Marshal(mocks.GenericTransaction(0))
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetTransactionFunc: func(_ context.Context, in *GetTransactionRequest, _ ...grpc.CallOption) (*GetTransactionResponse, error) {
					assert.Equal(t, mocks.ByteSlice(mocks.GenericIdentifier(0)), in.TransactionID)

					return &GetTransactionResponse{
						TransactionID: mocks.ByteSlice(mocks.GenericIdentifier(0)),
						Data:          testTransactionBytes,
					}, nil
				},
			},
		}

		got, err := index.Transaction(mocks.GenericIdentifier(0))

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericTransaction(0), got)
		}
	})

	codec := mocks.BaselineCodec(t)
	codec.UnmarshalFunc = cbor.Unmarshal

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			client: &apiMock{
				GetTransactionFunc: func(_ context.Context, in *GetTransactionRequest, _ ...grpc.CallOption) (*GetTransactionResponse, error) {
					assert.Equal(t, mocks.ByteSlice(mocks.GenericIdentifier(0)), in.TransactionID)

					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Transaction(mocks.GenericIdentifier(0))

		assert.Error(t, err)
	})
}

func TestIndex_Events(t *testing.T) {
	testEventsBytes, err := cbor.Marshal(mocks.GenericEvents(4))
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
					assert.Equal(t, convert.TypesToStrings(mocks.GenericEventTypes(2)), in.Types)

					return &GetEventsResponse{
						Height: mocks.GenericHeight,
						Types:  convert.TypesToStrings(mocks.GenericEventTypes(2)),
						Data:   testEventsBytes,
					}, nil
				},
			},
		}

		got, err := index.Events(mocks.GenericHeight, mocks.GenericEventTypes(2)...)

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericEvents(4), got)
		}
	})
	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetEventsFunc: func(_ context.Context, in *GetEventsRequest, _ ...grpc.CallOption) (*GetEventsResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)
					assert.Equal(t, convert.TypesToStrings(mocks.GenericEventTypes(2)), in.Types)

					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Events(mocks.GenericHeight, mocks.GenericEventTypes(2)...)

		assert.Error(t, err)
	})

	t.Run("handles invalid indexed data", func(t *testing.T) {
		t.Parallel()

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetEventsFunc: func(_ context.Context, in *GetEventsRequest, _ ...grpc.CallOption) (*GetEventsResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)
					assert.Equal(t, convert.TypesToStrings(mocks.GenericEventTypes(2)), in.Types)

					return &GetEventsResponse{
						Height: mocks.GenericHeight,
						Types:  convert.TypesToStrings(mocks.GenericEventTypes(2)),
						Data:   []byte(`invalid data`),
					}, nil
				},
			},
		}

		_, err := index.Events(mocks.GenericHeight, mocks.GenericEventTypes(2)...)

		assert.Error(t, err)
	})
}

func TestIndex_Seal(t *testing.T) {

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		testSealBytes, err := cbor.Marshal(mocks.GenericSeal(0))
		require.NoError(t, err)

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = cbor.Unmarshal

		index := Index{
			codec: codec,
			client: &apiMock{
				GetSealFunc: func(_ context.Context, in *GetSealRequest, _ ...grpc.CallOption) (*GetSealResponse, error) {
					assert.Equal(t, mocks.ByteSlice(mocks.GenericIdentifier(0)), in.SealID)

					return &GetSealResponse{
						SealID: mocks.ByteSlice(mocks.GenericIdentifier(0)),
						Data:   testSealBytes,
					}, nil
				},
			},
		}

		got, err := index.Seal(mocks.GenericIdentifier(0))
		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericSeal(0), got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				GetSealFunc: func(_ context.Context, in *GetSealRequest, _ ...grpc.CallOption) (*GetSealResponse, error) {
					assert.Equal(t, mocks.ByteSlice(mocks.GenericIdentifier(0)), in.SealID)

					return nil, mocks.GenericError
				},
			},
		}

		_, err := index.Seal(mocks.GenericIdentifier(0))
		assert.Error(t, err)
	})
}

func TestIndex_ListSealsForHeight(t *testing.T) {

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		testSealIDs := mocks.GenericIdentifiers(4)
		testSealBytes := make([][]byte, 0, len(testSealIDs))
		for _, sealID := range testSealIDs {
			id := make([]byte, len(sealID[:]))
			copy(id, sealID[:])
			testSealBytes = append(testSealBytes, id)
		}

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				ListSealsForHeightFunc: func(_ context.Context, in *ListSealsForHeightRequest, _ ...grpc.CallOption) (*ListSealsForHeightResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)

					return &ListSealsForHeightResponse{
						Height:   in.Height,
						SealsIDs: testSealBytes,
					}, nil
				},
			},
		}

		got, err := index.SealsByHeight(mocks.GenericHeight)
		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericIdentifiers(4), got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		t.Parallel()

		index := Index{
			codec: mocks.BaselineCodec(t),
			client: &apiMock{
				ListSealsForHeightFunc: func(_ context.Context, in *ListSealsForHeightRequest, _ ...grpc.CallOption) (*ListSealsForHeightResponse, error) {
					assert.Equal(t, mocks.GenericHeight, in.Height)

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
	GetTransactionFunc            func(ctx context.Context, in *GetTransactionRequest, opts ...grpc.CallOption) (*GetTransactionResponse, error)
	ListTransactionsForHeightFunc func(ctx context.Context, in *ListTransactionsForHeightRequest, opts ...grpc.CallOption) (*ListTransactionsForHeightResponse, error)
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

func (a *apiMock) GetTransaction(ctx context.Context, in *GetTransactionRequest, opts ...grpc.CallOption) (*GetTransactionResponse, error) {
	return a.GetTransactionFunc(ctx, in, opts...)
}

func (a *apiMock) ListTransactionsForHeight(ctx context.Context, in *ListTransactionsForHeightRequest, opts ...grpc.CallOption) (*ListTransactionsForHeightResponse, error) {
	return a.ListTransactionsForHeightFunc(ctx, in, opts...)
}

func (a *apiMock) GetSeal(ctx context.Context, in *GetSealRequest, opts ...grpc.CallOption) (*GetSealResponse, error) {
	return a.GetSealFunc(ctx, in, opts...)
}

func (a *apiMock) ListSealsForHeight(ctx context.Context, in *ListSealsForHeightRequest, opts ...grpc.CallOption) (*ListSealsForHeightResponse, error) {
	return a.ListSealsForHeightFunc(ctx, in, opts...)
}
