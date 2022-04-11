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

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewServer(t *testing.T) {
	index := mocks.BaselineReader(t)
	codec := mocks.BaselineCodec(t)

	s := NewServer(index, codec)

	require.NotNil(t, s)
	assert.NotNil(t, s.codec)
	assert.Equal(t, index, s.index)
	assert.Equal(t, codec, s.codec)
	assert.NotNil(t, s.validate)
}

func TestServer_GetFirst(t *testing.T) {
	tests := []struct {
		name     string
		mockErr  error
		wantRes  *GetFirstResponse
		checkErr require.ErrorAssertionFunc
	}{
		{
			name:    "nominal case",
			mockErr: nil,
			wantRes: &GetFirstResponse{
				Height: mocks.GenericHeight,
			},
			checkErr: require.NoError,
		},
		{
			name:     "error case",
			mockErr:  mocks.GenericError,
			wantRes:  nil,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.FirstFunc = func() (uint64, error) {
				return mocks.GenericHeight, test.mockErr
			}

			s := Server{
				index:    index,
				validate: validator.New(),
			}

			req := &GetFirstRequest{}

			got, err := s.GetFirst(context.Background(), req)

			test.checkErr(t, err)
			if err == nil {
				assert.Equal(t, test.wantRes, got)
			}
		})
	}
}

func TestServer_GetLast(t *testing.T) {

	tests := []struct {
		name     string
		mockErr  error
		wantRes  *GetLastResponse
		checkErr require.ErrorAssertionFunc
	}{
		{
			name:    "nominal case",
			mockErr: nil,
			wantRes: &GetLastResponse{
				Height: mocks.GenericHeight,
			},
			checkErr: require.NoError,
		},
		{
			name:     "error case",
			mockErr:  mocks.GenericError,
			wantRes:  nil,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.LastFunc = func() (uint64, error) {
				return mocks.GenericHeight, test.mockErr
			}

			s := Server{
				index:    index,
				validate: validator.New(),
			}

			req := &GetLastRequest{}

			got, err := s.GetLast(context.Background(), req)

			test.checkErr(t, err)
			if err == nil {
				assert.Equal(t, test.wantRes, got)
			}
		})
	}
}

func TestServer_GetHeightForBlock(t *testing.T) {
	blockID := mocks.GenericHeader.ID()
	tests := []struct {
		name        string
		req         *GetHeightForBlockRequest
		mockErr     error
		wantBlockID flow.Identifier
		checkErr    require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetHeightForBlockRequest{
				BlockID: mocks.ByteSlice(blockID),
			},
			mockErr:     nil,
			wantBlockID: blockID,
			checkErr:    require.NoError,
		},
		{
			name:     "handles missing block ID",
			req:      &GetHeightForBlockRequest{},
			checkErr: require.Error,
		},
		{
			name: "error handling",
			req: &GetHeightForBlockRequest{
				BlockID: mocks.ByteSlice(blockID),
			},
			mockErr:  mocks.GenericError,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.HeightForBlockFunc = func(_ flow.Identifier) (uint64, error) {
				return mocks.GenericHeight, test.mockErr
			}

			s := Server{
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetHeightForBlock(context.Background(), test.req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				assert.Equal(t, mocks.GenericHeight, got.Height)
				assert.Equal(t, test.wantBlockID[:], got.BlockID)
			}
		})
	}
}

func TestServer_GetCommit(t *testing.T) {
	tests := []struct {
		name       string
		req        *GetCommitRequest
		mockCommit flow.StateCommitment
		mockErr    error
		wantRes    *GetCommitResponse
		checkErr   require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetCommitRequest{
				Height: mocks.GenericHeight,
			},
			mockCommit: mocks.GenericCommit(0),
			mockErr:    nil,
			wantRes: &GetCommitResponse{
				Height: mocks.GenericHeight,
				Commit: mocks.ByteSlice(mocks.GenericCommit(0)),
			},
			checkErr: require.NoError,
		},
		{
			name: "error case",
			req: &GetCommitRequest{
				Height: mocks.GenericHeight,
			},
			mockCommit: flow.DummyStateCommitment,
			mockErr:    mocks.GenericError,
			wantRes:    nil,
			checkErr:   require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.CommitFunc = func(height uint64) (flow.StateCommitment, error) {
				assert.Equal(t, mocks.GenericHeight, height)

				return test.mockCommit, test.mockErr
			}

			s := Server{
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetCommit(context.Background(), test.req)

			test.checkErr(t, err)
			if err == nil {
				assert.Equal(t, test.wantRes, got)
			}
		})
	}
}

func TestServer_GetHeader(t *testing.T) {
	tests := []struct {
		name       string
		reqHeight  uint64
		mockHeader *flow.Header
		mockErr    error
		wantHeight uint64
		wantRes    *GetHeaderResponse
		checkErr   require.ErrorAssertionFunc
	}{
		{
			name:       "nominal case",
			reqHeight:  mocks.GenericHeight,
			mockHeader: mocks.GenericHeader,
			mockErr:    nil,
			wantHeight: mocks.GenericHeight,
			wantRes: &GetHeaderResponse{
				Height: mocks.GenericHeight,
				Data:   mocks.GenericBytes,
			},
			checkErr: require.NoError,
		},
		{
			name:       "error case",
			reqHeight:  mocks.GenericHeight,
			mockErr:    mocks.GenericError,
			wantHeight: mocks.GenericHeight,
			wantRes:    nil,
			checkErr:   require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			codec := mocks.BaselineCodec(t)
			codec.MarshalFunc = func(v interface{}) ([]byte, error) {
				assert.IsType(t, &flow.Header{}, v)

				return mocks.GenericBytes, nil
			}

			index := mocks.BaselineReader(t)
			index.HeaderFunc = func(height uint64) (*flow.Header, error) {
				assert.Equal(t, mocks.GenericHeight, height)

				return test.mockHeader, test.mockErr
			}

			s := Server{
				codec:    codec,
				index:    index,
				validate: validator.New(),
			}

			req := &GetHeaderRequest{
				Height: test.reqHeight,
			}
			got, err := s.GetHeader(context.Background(), req)

			test.checkErr(t, err)
			if err == nil {
				assert.Equal(t, test.wantRes, got)
			}
		})
	}
}

func TestServer_GetEvents(t *testing.T) {
	tests := []struct {
		name string

		reqHeight uint64
		reqTypes  []flow.EventType

		mockEvents []flow.Event
		mockErr    error

		wantHeight uint64
		wantTypes  []flow.EventType
		wantRes    *GetEventsResponse

		checkErr require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",

			reqHeight: mocks.GenericHeight,
			reqTypes:  mocks.GenericEventTypes(2),

			mockEvents: mocks.GenericEvents(4),
			mockErr:    nil,

			wantHeight: mocks.GenericHeight,
			wantTypes:  mocks.GenericEventTypes(2),
			wantRes: &GetEventsResponse{
				Height: mocks.GenericHeight,
				Types:  convert.TypesToStrings(mocks.GenericEventTypes(2)),
				Data:   mocks.GenericBytes,
			},

			checkErr: require.NoError,
		},
		{
			name: "error case",

			reqHeight: mocks.GenericHeight,
			reqTypes:  mocks.GenericEventTypes(2),

			mockEvents: mocks.GenericEvents(4),
			mockErr:    mocks.GenericError,

			wantHeight: mocks.GenericHeight,
			wantTypes:  mocks.GenericEventTypes(2),
			wantRes:    nil,

			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			codec := mocks.BaselineCodec(t)
			codec.MarshalFunc = func(v interface{}) ([]byte, error) {
				assert.IsType(t, []flow.Event{}, v)

				return mocks.GenericBytes, nil
			}

			index := mocks.BaselineReader(t)
			index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
				assert.Equal(t, mocks.GenericHeight, height)
				assert.Equal(t, test.wantTypes, types)

				return test.mockEvents, test.mockErr
			}

			s := Server{
				codec:    codec,
				index:    index,
				validate: validator.New(),
			}

			req := &GetEventsRequest{
				Height: test.reqHeight,
				Types:  convert.TypesToStrings(test.reqTypes),
			}
			got, err := s.GetEvents(context.Background(), req)

			test.checkErr(t, err)
			if err == nil {
				assert.Equal(t, test.wantRes, got)
			}
		})
	}
}

func TestServer_GetRegisterValues(t *testing.T) {
	tests := []struct {
		name     string
		req      *GetRegisterValuesRequest
		mockErr  error
		want     *GetRegisterValuesResponse
		checkErr require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetRegisterValuesRequest{
				Height: mocks.GenericHeight,
				Paths:  convert.PathsToBytes(mocks.GenericLedgerPaths(6)),
			},
			mockErr: nil,
			want: &GetRegisterValuesResponse{
				Height: mocks.GenericHeight,
				Paths:  convert.PathsToBytes(mocks.GenericLedgerPaths(6)),
				Values: convert.ValuesToBytes(mocks.GenericLedgerValues(6)),
			},
			checkErr: require.NoError,
		},
		{
			name: "handles missing paths",
			req: &GetRegisterValuesRequest{
				Height: mocks.GenericHeight,
			},
			want:     nil,
			checkErr: require.Error,
		},
		{
			name: "handles paths with invalid lengths",
			req: &GetRegisterValuesRequest{
				Height: mocks.GenericHeight,
				Paths:  [][]byte{mocks.GenericBytes},
			},
			want:     nil,
			checkErr: require.Error,
		},
		{
			name: "error case",
			req: &GetRegisterValuesRequest{
				Height: mocks.GenericHeight,
				Paths:  convert.PathsToBytes(mocks.GenericLedgerPaths(6)),
			},
			mockErr:  mocks.GenericError,
			want:     nil,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.ValuesFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
				if test.want != nil {
					assert.Equal(t, test.want.Height, height)
					assert.Equal(t, test.want.Paths, convert.PathsToBytes(paths))
				}

				return mocks.GenericLedgerValues(6), test.mockErr
			}

			s := Server{
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetRegisterValues(context.Background(), test.req)

			test.checkErr(t, err)
			if test.want != nil {
				require.NotNil(t, got)
				assert.Equal(t, test.want.Height, got.Height)
				assert.ElementsMatch(t, test.want.Values, got.Values)
				assert.ElementsMatch(t, test.want.Paths, got.Paths)
			}
		})
	}
}

func TestServer_GetCollection(t *testing.T) {
	collection := mocks.GenericCollection(0)

	tests := []struct {
		name           string
		req            *GetCollectionRequest
		mockCollection *flow.LightCollection
		mockErr        error
		checkErr       require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetCollectionRequest{
				CollectionID: mocks.ByteSlice(collection.ID()),
			},
			mockCollection: collection,
			checkErr:       require.NoError,
		},
		{
			name: "handles invalid collection ID",
			req: &GetCollectionRequest{
				CollectionID: mocks.GenericBytes,
			},
			mockCollection: collection,
			checkErr:       require.Error,
		},
		{
			name: "handles index failure",
			req: &GetCollectionRequest{
				CollectionID: mocks.ByteSlice(collection.ID()),
			},
			mockCollection: collection,
			mockErr:        mocks.GenericError,
			checkErr:       require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.CollectionFunc = func(id flow.Identifier) (*flow.LightCollection, error) {
				return test.mockCollection, test.mockErr
			}

			s := Server{
				codec:    mocks.BaselineCodec(t),
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetCollection(context.Background(), test.req)

			test.checkErr(t, err)
			if got != nil {
				require.NotNil(t, got)
				want := test.mockCollection.ID()
				assert.Equal(t, got.CollectionID, want[:])
				assert.NotEmpty(t, got.Data)
			}
		})
	}
}

func TestServer_ListCollectionsForHeight(t *testing.T) {
	tests := []struct {
		name            string
		reqHeight       uint64
		mockCollections []flow.Identifier
		mockErr         error
		checkErr        require.ErrorAssertionFunc
	}{
		{
			name:            "nominal case",
			reqHeight:       mocks.GenericHeight,
			mockCollections: mocks.GenericCollectionIDs(5),
			checkErr:        require.NoError,
		},
		{
			name:      "handles index failure",
			reqHeight: mocks.GenericHeight,
			mockErr:   mocks.GenericError,
			checkErr:  require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.CollectionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
				assert.Equal(t, mocks.GenericHeight, height)

				return test.mockCollections, test.mockErr
			}

			s := Server{
				index:    index,
				validate: validator.New(),
			}

			req := &ListCollectionsForHeightRequest{
				Height: mocks.GenericHeight,
			}
			got, err := s.ListCollectionsForHeight(context.Background(), req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				assert.Equal(t, got.Height, mocks.GenericHeight)
				assert.Len(t, got.CollectionIDs, len(test.mockCollections))
				for _, want := range test.mockCollections {
					assert.Contains(t, got.CollectionIDs, want[:])
				}
			}
		})
	}
}

func TestServer_GetGuarantee(t *testing.T) {
	guarantee := mocks.GenericGuarantee(0)

	tests := []struct {
		name          string
		req           *GetGuaranteeRequest
		mockErr       error
		wantGuarantee *flow.CollectionGuarantee
		checkErr      require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetGuaranteeRequest{
				CollectionID: mocks.ByteSlice(guarantee.CollectionID),
			},
			wantGuarantee: guarantee,
			checkErr:      require.NoError,
		},
		{
			name: "handles invalid collection ID",
			req: &GetGuaranteeRequest{
				CollectionID: mocks.GenericBytes,
			},
			checkErr: require.Error,
		},
		{
			name: "handles index failure",
			req: &GetGuaranteeRequest{
				CollectionID: mocks.ByteSlice(guarantee.CollectionID),
			},
			mockErr:  mocks.GenericError,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.GuaranteeFunc = func(_ flow.Identifier) (*flow.CollectionGuarantee, error) {
				return test.wantGuarantee, test.mockErr
			}

			s := Server{
				codec:    mocks.BaselineCodec(t),
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetGuarantee(context.Background(), test.req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				assert.Equal(t, got.CollectionID, test.req.CollectionID)
				assert.NotEmpty(t, got.Data)
			}
		})
	}
}

func TestServer_GetTransaction(t *testing.T) {
	tx := mocks.GenericTransaction(0)

	tests := []struct {
		name            string
		req             *GetTransactionRequest
		mockErr         error
		wantTransaction *flow.TransactionBody
		checkErr        require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetTransactionRequest{
				TransactionID: mocks.ByteSlice(tx.ID()),
			},
			wantTransaction: tx,
			checkErr:        require.NoError,
		},
		{
			name: "handles invalid transaction ID",
			req: &GetTransactionRequest{
				TransactionID: mocks.GenericBytes,
			},
			checkErr: require.Error,
		},
		{
			name: "handles index failure",
			req: &GetTransactionRequest{
				TransactionID: mocks.ByteSlice(tx.ID()),
			},
			mockErr:  mocks.GenericError,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.TransactionFunc = func(_ flow.Identifier) (*flow.TransactionBody, error) {
				return test.wantTransaction, test.mockErr
			}

			s := Server{
				codec:    mocks.BaselineCodec(t),
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetTransaction(context.Background(), test.req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				want := tx.ID()
				assert.Equal(t, got.TransactionID, want[:])
				assert.NotEmpty(t, got.Data)
			}
		})
	}
}

func TestServer_GetHeightForTransaction(t *testing.T) {
	blockID := mocks.GenericHeader.ID()

	tests := []struct {
		name     string
		req      *GetHeightForTransactionRequest
		mockErr  error
		wantTxID flow.Identifier
		checkErr require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetHeightForTransactionRequest{
				TransactionID: mocks.ByteSlice(blockID),
			},
			mockErr:  nil,
			wantTxID: blockID,
			checkErr: require.NoError,
		},
		{
			name: "handles invalid transaction ID",
			req: &GetHeightForTransactionRequest{
				TransactionID: mocks.GenericBytes,
			},
			checkErr: require.Error,
		},
		{
			name: "error handling",
			req: &GetHeightForTransactionRequest{
				TransactionID: mocks.ByteSlice(blockID),
			},
			mockErr:  mocks.GenericError,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.HeightForTransactionFunc = func(blockID flow.Identifier) (uint64, error) {
				return mocks.GenericHeight, test.mockErr
			}

			s := Server{
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetHeightForTransaction(context.Background(), test.req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				assert.Equal(t, mocks.GenericHeight, got.Height)
				assert.Equal(t, test.wantTxID[:], got.TransactionID)
			}
		})
	}
}

func TestServer_ListTransactionsForHeight(t *testing.T) {
	tests := []struct {
		name             string
		reqHeight        uint64
		mockTransactions []flow.Identifier
		mockErr          error
		checkErr         require.ErrorAssertionFunc
	}{
		{
			name:             "nominal case",
			reqHeight:        mocks.GenericHeight,
			mockTransactions: mocks.GenericTransactionIDs(5),
			checkErr:         require.NoError,
		},
		{
			name:      "handles index failure",
			reqHeight: mocks.GenericHeight,
			mockErr:   mocks.GenericError,
			checkErr:  require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
				return test.mockTransactions, test.mockErr
			}

			s := Server{
				index:    index,
				validate: validator.New(),
			}

			req := &ListTransactionsForHeightRequest{
				Height: mocks.GenericHeight,
			}
			got, err := s.ListTransactionsForHeight(context.Background(), req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				assert.Equal(t, got.Height, mocks.GenericHeight)
				assert.Len(t, got.TransactionIDs, len(test.mockTransactions))
				for _, want := range test.mockTransactions {
					assert.Contains(t, got.TransactionIDs, want[:])
				}
			}
		})
	}
}

func TestServer_GetResult(t *testing.T) {
	result := mocks.GenericResult(0)

	tests := []struct {
		name       string
		req        *GetResultRequest
		mockResult *flow.TransactionResult
		mockErr    error
		checkErr   require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetResultRequest{
				TransactionID: mocks.ByteSlice(result.TransactionID),
			},
			mockResult: result,
			checkErr:   require.NoError,
		},
		{
			name: "handles invalid transaction ID",
			req: &GetResultRequest{
				TransactionID: mocks.GenericBytes,
			},
			checkErr: require.Error,
		},
		{
			name: "handles index failure",
			req: &GetResultRequest{
				TransactionID: mocks.ByteSlice(result.TransactionID),
			},
			mockErr:  mocks.GenericError,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.ResultFunc = func(_ flow.Identifier) (*flow.TransactionResult, error) {
				return test.mockResult, test.mockErr
			}

			s := Server{
				codec:    mocks.BaselineCodec(t),
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetResult(context.Background(), test.req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				assert.Equal(t, got.TransactionID, mocks.ByteSlice(result.TransactionID))
				assert.NotEmpty(t, got.Data)
			}
		})
	}
}

func TestServer_GetSeal(t *testing.T) {
	seal := mocks.GenericSeal(0)

	tests := []struct {
		name     string
		req      *GetSealRequest
		mockSeal *flow.Seal
		mockErr  error
		checkErr require.ErrorAssertionFunc
	}{
		{
			name: "nominal case",
			req: &GetSealRequest{
				SealID: mocks.ByteSlice(seal.ID()),
			},
			mockSeal: seal,
			checkErr: require.NoError,
		},
		{
			name: "handles invalid seal ID",
			req: &GetSealRequest{
				SealID: mocks.GenericBytes,
			},
			checkErr: require.Error,
		},
		{
			name: "handles index failure",
			req: &GetSealRequest{
				SealID: mocks.ByteSlice(seal.ID()),
			},
			mockErr:  mocks.GenericError,
			checkErr: require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.SealFunc = func(sealID flow.Identifier) (*flow.Seal, error) {
				return test.mockSeal, test.mockErr
			}

			s := Server{
				codec:    mocks.BaselineCodec(t),
				index:    index,
				validate: validator.New(),
			}

			got, err := s.GetSeal(context.Background(), test.req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				assert.Equal(t, got.SealID, test.req.SealID)
				assert.NotEmpty(t, got.Data)
			}
		})
	}
}

func TestServer_ListSealsForHeight(t *testing.T) {
	sealIDs := mocks.GenericSealIDs(5)

	tests := []struct {
		name      string
		reqHeight uint64
		mockSeals []flow.Identifier
		mockErr   error
		checkErr  require.ErrorAssertionFunc
	}{
		{
			name:      "nominal case",
			reqHeight: mocks.GenericHeight,
			mockSeals: sealIDs,
			checkErr:  require.NoError,
		},
		{
			name:      "handles index failure",
			reqHeight: mocks.GenericHeight,
			mockErr:   mocks.GenericError,
			checkErr:  require.Error,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.SealsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
				assert.Equal(t, mocks.GenericHeight, height)

				return test.mockSeals, test.mockErr
			}

			s := Server{
				codec:    mocks.BaselineCodec(t),
				index:    index,
				validate: validator.New(),
			}

			req := ListSealsForHeightRequest{
				Height: mocks.GenericHeight,
			}

			got, err := s.ListSealsForHeight(context.Background(), &req)

			test.checkErr(t, err)
			if err == nil {
				require.NotNil(t, got)
				assert.Equal(t, got.Height, test.reqHeight)
				assert.Len(t, got.SealIDs, len(test.mockSeals))
				for _, want := range test.mockSeals {
					assert.Contains(t, got.SealIDs, want[:])
				}
			}
		})
	}
}
