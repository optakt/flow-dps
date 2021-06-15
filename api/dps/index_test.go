// Copyright 2021 Alvalor S.A.
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

	t.Run("handles indexing failure", func(t *testing.T) {
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

	t.Run("handles indexing failure", func(t *testing.T) {
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

	t.Run("handles indexing failures", func(t *testing.T) {
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

	t.Run("handles marshalling failures", func(t *testing.T) {
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

	t.Run("handles indexing failures", func(t *testing.T) {
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

	t.Run("handles indexing failures", func(t *testing.T) {
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
	path1, _ := ledger.ToPath([]byte("aac513eb1a0457700ac3fa8d292513e1"))
	path2, _ := ledger.ToPath([]byte("bbc513eb1a5465415ac3fa8d292514f2"))
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

	t.Run("handles indexing failures", func(t *testing.T) {
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

	t.Run("handles indexing failures", func(t *testing.T) {
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

type apiMock struct {
	GetFirstFunc     func(ctx context.Context, in *GetFirstRequest, opts ...grpc.CallOption) (*GetFirstResponse, error)
	GetLastFunc      func(ctx context.Context, in *GetLastRequest, opts ...grpc.CallOption) (*GetLastResponse, error)
	GetHeaderFunc    func(ctx context.Context, in *GetHeaderRequest, opts ...grpc.CallOption) (*GetHeaderResponse, error)
	GetCommitFunc    func(ctx context.Context, in *GetCommitRequest, opts ...grpc.CallOption) (*GetCommitResponse, error)
	GetEventsFunc    func(ctx context.Context, in *GetEventsRequest, opts ...grpc.CallOption) (*GetEventsResponse, error)
	GetRegistersFunc func(ctx context.Context, in *GetRegistersRequest, opts ...grpc.CallOption) (*GetRegistersResponse, error)
	GetHeightFunc    func(ctx context.Context, in *GetHeightRequest, opts ...grpc.CallOption) (*GetHeightResponse, error)
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
