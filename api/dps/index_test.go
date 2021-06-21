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

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestIndexFromAPI(t *testing.T) {
	mock := &apiMock{}

	index := IndexFromAPI(mock)

	if assert.NotNil(t, index) {
		assert.Equal(t, mock, index.client)
	}
}

func TestIndex_First(t *testing.T) {
	testHeight := uint64(42)

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetFirstFunc: func(_ context.Context, in *GetFirstRequest, _ ...grpc.CallOption) (*GetFirstResponse, error) {
					assert.NotNil(t, in)

					return &GetFirstResponse{
						Height: testHeight,
					}, nil
				},
			},
		}

		got, err := index.First()

		if assert.NoError(t, err) {
			assert.Equal(t, testHeight, got)
		}
	})

	t.Run("handles index failure", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetFirstFunc: func(_ context.Context, in *GetFirstRequest, _ ...grpc.CallOption) (*GetFirstResponse, error) {
					assert.NotNil(t, in)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.First()
		assert.Error(t, err)
	})
}

func TestIndex_Last(t *testing.T) {
	testHeight := uint64(42)

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetLastFunc: func(_ context.Context, in *GetLastRequest, _ ...grpc.CallOption) (*GetLastResponse, error) {
					assert.NotNil(t, in)

					return &GetLastResponse{
						Height: testHeight,
					}, nil
				},
			},
		}

		got, err := index.Last()

		if assert.NoError(t, err) {
			assert.Equal(t, testHeight, got)
		}
	})

	t.Run("handles index failure", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetLastFunc: func(_ context.Context, in *GetLastRequest, _ ...grpc.CallOption) (*GetLastResponse, error) {
					assert.NotNil(t, in)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.Last()
		assert.Error(t, err)
	})

}

func TestIndex_Header(t *testing.T) {
	testHeader := &flow.Header{
		ChainID: dps.FlowTestnet,
		Height:  42,
	}

	testHeaderB, err := cbor.Marshal(testHeader)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetHeaderFunc: func(_ context.Context, in *GetHeaderRequest, _ ...grpc.CallOption) (*GetHeaderResponse, error) {
					assert.Equal(t, testHeader.Height, in.Height)

					return &GetHeaderResponse{
						Height: testHeader.Height,
						Data:   testHeaderB,
					}, nil
				},
			},
		}

		got, err := index.Header(testHeader.Height)

		if assert.NoError(t, err) {
			assert.Equal(t, testHeader, got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetHeaderFunc: func(_ context.Context, in *GetHeaderRequest, _ ...grpc.CallOption) (*GetHeaderResponse, error) {
					assert.Equal(t, testHeader.Height, in.Height)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.Header(testHeader.Height)

		assert.Error(t, err)
	})

	t.Run("handles decoding failures", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetHeaderFunc: func(_ context.Context, in *GetHeaderRequest, _ ...grpc.CallOption) (*GetHeaderResponse, error) {
					assert.Equal(t, testHeader.Height, in.Height)

					return &GetHeaderResponse{
						Height: testHeader.Height,
						Data:   []byte(`invalid data`),
					}, nil
				},
			},
		}

		_, err := index.Header(testHeader.Height)

		assert.Error(t, err)
	})
}

func TestIndex_Commit(t *testing.T) {
	testHeight := uint64(42)
	testCommit, err := flow.ToStateCommitment([]byte("07018030187ecf04945f35f1e33a89dc"))
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetCommitFunc: func(_ context.Context, in *GetCommitRequest, _ ...grpc.CallOption) (*GetCommitResponse, error) {
					assert.Equal(t, testHeight, in.Height)

					return &GetCommitResponse{
						Height: testHeight,
						Commit: testCommit[:],
					}, nil
				},
			},
		}

		got, err := index.Commit(testHeight)

		if assert.NoError(t, err) {
			assert.Equal(t, testCommit, got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetCommitFunc: func(_ context.Context, in *GetCommitRequest, _ ...grpc.CallOption) (*GetCommitResponse, error) {
					assert.Equal(t, testHeight, in.Height)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.Commit(testHeight)

		assert.Error(t, err)
	})

	t.Run("handles invalid indexed data", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetCommitFunc: func(_ context.Context, in *GetCommitRequest, _ ...grpc.CallOption) (*GetCommitResponse, error) {
					assert.Equal(t, testHeight, in.Height)

					return &GetCommitResponse{
						Height: testHeight,
						Commit: []byte(`not a commit`),
					}, nil
				},
			},
		}

		_, err := index.Commit(testHeight)

		assert.Error(t, err)
	})
}

func TestIndex_Events(t *testing.T) {
	testHeight := uint64(42)
	testTypes := []flow.EventType{"deposit", "withdrawal"}
	testEvents := []flow.Event{
		{Type: "deposit"},
		{Type: "withdrawal"},
	}
	testEventsB, err := cbor.Marshal(testEvents)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetEventsFunc: func(_ context.Context, in *GetEventsRequest, _ ...grpc.CallOption) (*GetEventsResponse, error) {
					assert.Equal(t, testHeight, in.Height)
					assert.Equal(t, convert.TypesToStrings(testTypes), in.Types)

					return &GetEventsResponse{
						Height: testHeight,
						Types:  convert.TypesToStrings(testTypes),
						Data:   testEventsB,
					}, nil
				},
			},
		}

		got, err := index.Events(testHeight, testTypes...)

		if assert.NoError(t, err) {
			assert.Equal(t, testEvents, got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetEventsFunc: func(_ context.Context, in *GetEventsRequest, _ ...grpc.CallOption) (*GetEventsResponse, error) {
					assert.Equal(t, testHeight, in.Height)
					assert.Equal(t, convert.TypesToStrings(testTypes), in.Types)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.Events(testHeight, testTypes...)

		assert.Error(t, err)
	})

	t.Run("handles invalid indexed data", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetEventsFunc: func(_ context.Context, in *GetEventsRequest, _ ...grpc.CallOption) (*GetEventsResponse, error) {
					assert.Equal(t, testHeight, in.Height)
					assert.Equal(t, convert.TypesToStrings(testTypes), in.Types)

					return &GetEventsResponse{
						Height: testHeight,
						Types:  convert.TypesToStrings(testTypes),
						Data:   []byte(`invalid data`),
					}, nil
				},
			},
		}

		_, err := index.Events(testHeight, testTypes...)

		assert.Error(t, err)
	})
}

func TestIndex_Registers(t *testing.T) {
	testHeight := uint64(42)
	path1 := ledger.Path{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1}
	path2 := ledger.Path{0xbb, 0xc5, 0x13, 0xeb, 0x1a, 0x54, 0x65, 0x41, 0x5a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x14, 0xf2}
	testPaths := []ledger.Path{path1, path2}
	testValues := []ledger.Value{ledger.Value(`test1`), ledger.Value(`test2`)}

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetRegistersFunc: func(_ context.Context, in *GetRegistersRequest, _ ...grpc.CallOption) (*GetRegistersResponse, error) {
					if assert.Len(t, in.Paths, 2) {
						assert.Equal(t, path1[:], in.Paths[0])
						assert.Equal(t, path2[:], in.Paths[1])
					}
					assert.Equal(t, in.Height, testHeight)

					return &GetRegistersResponse{
						Height: testHeight,
						Paths:  convert.PathsToBytes(testPaths),
						Values: convert.ValuesToBytes(testValues),
					}, nil
				},
			},
		}

		got, err := index.Registers(testHeight, testPaths)

		if assert.NoError(t, err) {
			assert.Equal(t, testValues, got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetRegistersFunc: func(_ context.Context, in *GetRegistersRequest, _ ...grpc.CallOption) (*GetRegistersResponse, error) {
					if assert.Len(t, in.Paths, 2) {
						assert.Equal(t, path1[:], in.Paths[0])
						assert.Equal(t, path2[:], in.Paths[1])
					}
					assert.Equal(t, in.Height, testHeight)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.Registers(testHeight, testPaths)

		assert.Error(t, err)
	})
}

func TestIndex_Height(t *testing.T) {
	testHeight := uint64(42)
	testBlockID, err := flow.HexStringToIdentifier("98827808c61af6b29c7f16071e69a9bbfba40d0f96b572ce23994b3aa605c7c2")
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetHeightFunc: func(_ context.Context, in *GetHeightRequest, _ ...grpc.CallOption) (*GetHeightResponse, error) {
					assert.Equal(t, testBlockID[:], in.BlockID)

					return &GetHeightResponse{
						BlockID: testBlockID[:],
						Height:  testHeight,
					}, nil
				},
			},
		}

		got, err := index.Height(testBlockID)

		if assert.NoError(t, err) {
			assert.Equal(t, testHeight, got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetHeightFunc: func(_ context.Context, in *GetHeightRequest, _ ...grpc.CallOption) (*GetHeightResponse, error) {
					assert.Equal(t, testBlockID[:], in.BlockID)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.Height(testBlockID)

		assert.Error(t, err)
	})
}

func TestIndex_Transaction(t *testing.T) {
	testTransactionID := flow.Identifier{0x98, 0x82, 0x78, 0x08, 0xc6, 0x1a, 0xf6, 0xb2, 0x9c, 0x7f, 0x16, 0x07, 0x1e, 0x69, 0xa9, 0xbb, 0xfb, 0xa4, 0x0d, 0x0f, 0x96, 0xb5, 0x72, 0xce, 0x23, 0x99, 0x4b, 0x3a, 0xa6, 0x05, 0xc7, 0xc2}
	testTransaction := flow.Transaction{
		TransactionBody: flow.TransactionBody{
			ReferenceBlockID: flow.Identifier{0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a},
			Payer:            flow.Address{0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a},
		},
	}
	testTransactionB, err := cbor.Marshal(testTransaction)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetTransactionFunc: func(_ context.Context, in *GetTransactionRequest, _ ...grpc.CallOption) (*GetTransactionResponse, error) {
					assert.Equal(t, testTransactionID[:], in.TransactionID)

					return &GetTransactionResponse{
						TransactionID:   testTransactionID[:],
						TransactionData: testTransactionB,
					}, nil
				},
			},
		}

		got, err := index.Transaction(testTransactionID)

		if assert.NoError(t, err) {
			assert.Equal(t, &testTransaction, got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetTransactionFunc: func(_ context.Context, in *GetTransactionRequest, _ ...grpc.CallOption) (*GetTransactionResponse, error) {
					assert.Equal(t, testTransactionID[:], in.TransactionID)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.Transaction(testTransactionID)

		assert.Error(t, err)
	})
}

func TestIndex_Collection(t *testing.T) {
	testTransactionID := flow.Identifier{0xd4, 0x7b, 0x1b, 0xf7, 0xf3, 0x7e, 0x19, 0x2c, 0xf8, 0x3d, 0x2b, 0xee, 0x3f, 0x63, 0x32, 0xb0, 0xd9, 0xb1, 0x5c, 0xa, 0xa7, 0x66, 0xd, 0x1e, 0x53, 0x22, 0xea, 0x96, 0x46, 0x67, 0xb3, 0x33}
	testCollectionID := flow.Identifier{0x98, 0x82, 0x78, 0x08, 0xc6, 0x1a, 0xf6, 0xb2, 0x9c, 0x7f, 0x16, 0x07, 0x1e, 0x69, 0xa9, 0xbb, 0xfb, 0xa4, 0x0d, 0x0f, 0x96, 0xb5, 0x72, 0xce, 0x23, 0x99, 0x4b, 0x3a, 0xa6, 0x05, 0xc7, 0xc2}
	testCollection := flow.LightCollection{Transactions: []flow.Identifier{testTransactionID, testTransactionID, testTransactionID, testTransactionID, testTransactionID}}

	t.Run("nominal case", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetCollectionFunc: func(_ context.Context, in *GetCollectionRequest, _ ...grpc.CallOption) (*GetCollectionResponse, error) {
					assert.Equal(t, testCollectionID[:], in.CollectionID)

					var transactionIDs [][]byte
					for _, transaction := range testCollection.Transactions {
						transactionIDs = append(transactionIDs, transaction[:])
					}

					return &GetCollectionResponse{
						CollectionID:   testCollectionID[:],
						TransactionIDs: transactionIDs,
					}, nil
				},
			},
		}

		got, err := index.Collection(testCollectionID)

		if assert.NoError(t, err) {
			assert.Equal(t, &testCollection, got)
		}
	})

	t.Run("handles index failures", func(t *testing.T) {
		index := Index{
			client: &apiMock{
				GetCollectionFunc: func(_ context.Context, in *GetCollectionRequest, _ ...grpc.CallOption) (*GetCollectionResponse, error) {
					assert.Equal(t, testCollectionID[:], in.CollectionID)

					return nil, mocks.DummyError
				},
			},
		}

		_, err := index.Collection(testCollectionID)

		assert.Error(t, err)
	})
}

type apiMock struct {
	GetFirstFunc        func(ctx context.Context, in *GetFirstRequest, opts ...grpc.CallOption) (*GetFirstResponse, error)
	GetLastFunc         func(ctx context.Context, in *GetLastRequest, opts ...grpc.CallOption) (*GetLastResponse, error)
	GetHeaderFunc       func(ctx context.Context, in *GetHeaderRequest, opts ...grpc.CallOption) (*GetHeaderResponse, error)
	GetCommitFunc       func(ctx context.Context, in *GetCommitRequest, opts ...grpc.CallOption) (*GetCommitResponse, error)
	GetEventsFunc       func(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (*GetEventsResponse, error)
	GetRegistersFunc    func(ctx context.Context, in *GetRegistersRequest, opts ...grpc.CallOption) (*GetRegistersResponse, error)
	GetHeightFunc       func(ctx context.Context, in *GetHeightRequest, opts ...grpc.CallOption) (*GetHeightResponse, error)
	GetTransactionFunc  func(ctx context.Context, in *GetTransactionRequest, opts ...grpc.CallOption) (*GetTransactionResponse, error)
	GetTransactionsFunc func(ctx context.Context, in *GetTransactionsRequest, opts ...grpc.CallOption) (*GetTransactionsResponse, error)
	GetCollectionFunc   func(ctx context.Context, in *GetCollectionRequest, opts ...grpc.CallOption) (*GetCollectionResponse, error)
	GetCollectionsFunc  func(ctx context.Context, in *GetCollectionsRequest, opts ...grpc.CallOption) (*GetCollectionsResponse, error)
}

func (a *apiMock) GetFirst(ctx context.Context, in *GetFirstRequest, opts ...grpc.CallOption) (*GetFirstResponse, error) {
	return a.GetFirstFunc(ctx, in, opts...)
}

func (a *apiMock) GetLast(ctx context.Context, in *GetLastRequest, opts ...grpc.CallOption) (*GetLastResponse, error) {
	return a.GetLastFunc(ctx, in, opts...)
}

func (a *apiMock) GetHeader(ctx context.Context, in *GetHeaderRequest, opts ...grpc.CallOption) (*GetHeaderResponse, error) {
	return a.GetHeaderFunc(ctx, in, opts...)
}

func (a *apiMock) GetCommit(ctx context.Context, in *GetCommitRequest, opts ...grpc.CallOption) (*GetCommitResponse, error) {
	return a.GetCommitFunc(ctx, in, opts...)
}

func (a *apiMock) GetEvents(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (*GetEventsResponse, error) {
	return a.GetEventsFunc(ctx, in, opts...)
}

func (a *apiMock) GetRegisters(ctx context.Context, in *GetRegistersRequest, opts ...grpc.CallOption) (*GetRegistersResponse, error) {
	return a.GetRegistersFunc(ctx, in, opts...)
}

func (a *apiMock) GetHeight(ctx context.Context, in *GetHeightRequest, opts ...grpc.CallOption) (*GetHeightResponse, error) {
	return a.GetHeightFunc(ctx, in, opts...)
}

func (a *apiMock) GetTransaction(ctx context.Context, in *GetTransactionRequest, opts ...grpc.CallOption) (*GetTransactionResponse, error) {
	return a.GetTransactionFunc(ctx, in, opts...)
}

func (a *apiMock) GetTransactions(ctx context.Context, in *GetTransactionsRequest, opts ...grpc.CallOption) (*GetTransactionsResponse, error) {
	return a.GetTransactionsFunc(ctx, in, opts...)
}

func (a *apiMock) GetCollection(ctx context.Context, in *GetCollectionRequest, opts ...grpc.CallOption) (*GetCollectionResponse, error) {
	return a.GetCollectionFunc(ctx, in, opts...)
}

func (a *apiMock) GetCollections(ctx context.Context, in *GetCollectionsRequest, opts ...grpc.CallOption) (*GetCollectionsResponse, error) {
	return a.GetCollectionsFunc(ctx, in, opts...)
}
