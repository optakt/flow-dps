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
	"github.com/optakt/flow-dps/models/convert"
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

func TestServer_GetFirst(t *testing.T) {
	tests := []struct {
		name string

		mockErr error

		wantRes *GetFirstResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			mockErr: nil,

			wantRes: &GetFirstResponse{
				Height: mocks.GenericHeight,
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			mockErr: mocks.GenericError,

			wantRes: nil,

			checkErr: assert.Error,
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

			s := Server{index: index}

			req := &GetFirstRequest{}

			gotRes, gotErr := s.GetFirst(context.Background(), req)
			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetLast(t *testing.T) {

	tests := []struct {
		name string

		mockErr error

		wantRes *GetLastResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			mockErr: nil,

			wantRes: &GetLastResponse{
				Height: mocks.GenericHeight,
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			mockErr: mocks.GenericError,

			wantRes: nil,

			checkErr: assert.Error,
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

			s := Server{index: index}

			req := &GetLastRequest{}

			gotRes, gotErr := s.GetLast(context.Background(), req)
			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetHeightForBlock(t *testing.T) {
	tests := []struct {
		name string

		reqBlockID flow.Identifier

		mockErr error

		wantBlockID flow.Identifier

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqBlockID: mocks.GenericIdentifier(0),

			mockErr: nil,

			wantBlockID: mocks.GenericIdentifier(0),

			checkErr: assert.NoError,
		},
		{
			name: "error handling",

			reqBlockID: mocks.GenericIdentifier(0),

			mockErr: mocks.GenericError,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.HeightForBlockFunc = func(blockID flow.Identifier) (uint64, error) {
				return mocks.GenericHeight, test.mockErr
			}

			s := Server{index: index}

			req := &GetHeightForBlockRequest{
				BlockID: mocks.ByteSlice(mocks.GenericIdentifier(0)),
			}
			gotRes, gotErr := s.GetHeightForBlock(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, mocks.GenericHeight, gotRes.Height)
				assert.Equal(t, test.wantBlockID[:], gotRes.BlockID)
			}
		})
	}
}

func TestServer_GetCommit(t *testing.T) {
	tests := []struct {
		name string

		mockCommit flow.StateCommitment
		mockErr    error

		wantRes *GetCommitResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			mockCommit: mocks.GenericCommit(0),
			mockErr:    nil,

			wantRes: &GetCommitResponse{
				Height: mocks.GenericHeight,
				Commit: mocks.ByteSlice(mocks.GenericCommit(0)),
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			mockCommit: flow.StateCommitment{},
			mockErr:    mocks.GenericError,

			wantRes: nil,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var gotHeight uint64
			index := mocks.BaselineReader(t)
			index.CommitFunc = func(height uint64) (flow.StateCommitment, error) {
				gotHeight = height
				return test.mockCommit, test.mockErr
			}

			s := Server{index: index}

			req := &GetCommitRequest{
				Height: mocks.GenericHeight,
			}
			gotRes, gotErr := s.GetCommit(context.Background(), req)

			test.checkErr(t, gotErr)
			assert.Equal(t, mocks.GenericHeight, gotHeight)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetHeader(t *testing.T) {
	tests := []struct {
		name string

		reqHeight uint64

		mockHeader *flow.Header
		mockErr    error

		wantHeight uint64
		wantRes    *GetHeaderResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqHeight: mocks.GenericHeight,

			mockHeader: mocks.GenericHeader,
			mockErr:    nil,

			wantHeight: mocks.GenericHeight,
			wantRes: &GetHeaderResponse{
				Height: mocks.GenericHeight,
				Data:   mocks.GenericBytes,
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			reqHeight: mocks.GenericHeight,

			mockErr: mocks.GenericError,

			wantHeight: mocks.GenericHeight,
			wantRes:    nil,

			checkErr: assert.Error,
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

			var gotHeight uint64
			index := mocks.BaselineReader(t)
			index.HeaderFunc = func(height uint64) (*flow.Header, error) {
				gotHeight = height
				return test.mockHeader, test.mockErr
			}

			s := Server{
				codec: codec,
				index: index,
			}

			req := &GetHeaderRequest{
				Height: test.reqHeight,
			}
			gotRes, gotErr := s.GetHeader(context.Background(), req)

			test.checkErr(t, gotErr)
			assert.Equal(t, mocks.GenericHeight, gotHeight)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
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

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

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

			checkErr: assert.NoError,
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

			checkErr: assert.Error,
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

			var gotHeight uint64
			var gotTypes []flow.EventType
			index := mocks.BaselineReader(t)
			index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
				gotHeight = height
				gotTypes = types
				return test.mockEvents, test.mockErr
			}

			s := Server{
				codec: codec,
				index: index,
			}

			req := &GetEventsRequest{
				Height: test.reqHeight,
				Types:  convert.TypesToStrings(test.reqTypes),
			}
			gotRes, gotErr := s.GetEvents(context.Background(), req)

			test.checkErr(t, gotErr)
			assert.Equal(t, mocks.GenericHeight, gotHeight)
			assert.Equal(t, test.wantTypes, gotTypes)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetRegisterValues(t *testing.T) {
	tests := []struct {
		name string

		reqHeight uint64
		reqPaths  []ledger.Path

		mockValues []ledger.Value
		mockErr    error

		wantHeight uint64
		wantPaths  []ledger.Path
		wantRes    *GetRegisterValuesResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqHeight: mocks.GenericHeight,
			reqPaths:  mocks.GenericLedgerPaths(6),

			mockValues: mocks.GenericLedgerValues(6),
			mockErr:    nil,

			wantHeight: mocks.GenericHeight,
			wantPaths:  mocks.GenericLedgerPaths(6),
			wantRes: &GetRegisterValuesResponse{
				Height: mocks.GenericHeight,
				Paths:  convert.PathsToBytes(mocks.GenericLedgerPaths(6)),
				Values: convert.ValuesToBytes(mocks.GenericLedgerValues(6)),
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			reqHeight: mocks.GenericHeight,
			reqPaths:  mocks.GenericLedgerPaths(6),

			mockValues: mocks.GenericLedgerValues(6),
			mockErr:    mocks.GenericError,

			wantHeight: mocks.GenericHeight,
			wantPaths:  mocks.GenericLedgerPaths(6),
			wantRes:    nil,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var gotHeight uint64
			var gotPaths []ledger.Path
			index := mocks.BaselineReader(t)
			index.ValuesFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
				gotHeight = height
				gotPaths = paths
				return test.mockValues, test.mockErr
			}

			s := Server{index: index}

			req := &GetRegisterValuesRequest{
				Height: test.reqHeight,
				Paths:  convert.PathsToBytes(test.reqPaths),
			}
			gotRes, gotErr := s.GetRegisterValues(context.Background(), req)

			test.checkErr(t, gotErr)
			assert.Equal(t, mocks.GenericHeight, gotHeight)
			assert.Equal(t, test.wantPaths, gotPaths)
			if gotErr == nil {
				assert.Equal(t, test.wantRes.Height, gotRes.Height)
				assert.EqualValues(t, test.wantRes.Paths, gotRes.Paths)
				assert.EqualValues(t, test.wantRes.Values, gotRes.Values)
			}
		})
	}
}

func TestServer_GetCollection(t *testing.T) {
	tests := []struct {
		name string

		reqCollectionID flow.Identifier

		mockCollection *flow.LightCollection
		mockErr        error

		wantCollection *flow.LightCollection

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqCollectionID: mocks.GenericIdentifier(0),

			mockCollection: mocks.GenericCollection(0),

			wantCollection: mocks.GenericCollection(0),
			checkErr:       assert.NoError,
		},
		{
			name: "handles index failure",

			reqCollectionID: mocks.GenericIdentifier(0),

			mockErr: mocks.GenericError,

			checkErr: assert.Error,
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

			s := Server{index: index}

			req := &GetCollectionRequest{
				CollectionID: mocks.ByteSlice(mocks.GenericIdentifier(0)),
			}
			gotRes, gotErr := s.GetCollection(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.CollectionID, mocks.ByteSlice(mocks.GenericIdentifier(0)))
				assert.NotEmpty(t, gotRes.Data)
			}
		})
	}
}

func TestServer_GetTransaction(t *testing.T) {
	tests := []struct {
		name string

		reqTransactionID flow.Identifier

		mockTransaction *flow.TransactionBody
		mockErr         error

		wantTransaction *flow.TransactionBody

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqTransactionID: mocks.GenericIdentifier(0),

			mockTransaction: mocks.GenericTransaction(0),

			wantTransaction: mocks.GenericTransaction(0),
			checkErr:        assert.NoError,
		},
		{
			name: "handles index failure",

			reqTransactionID: mocks.GenericIdentifier(0),

			mockErr: mocks.GenericError,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.TransactionFunc = func(transactionID flow.Identifier) (*flow.TransactionBody, error) {
				return test.mockTransaction, test.mockErr
			}

			s := Server{
				codec: mocks.BaselineCodec(t),
				index: index,
			}

			req := &GetTransactionRequest{
				TransactionID: mocks.ByteSlice(mocks.GenericIdentifier(0)),
			}
			gotRes, gotErr := s.GetTransaction(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.TransactionID, mocks.ByteSlice(mocks.GenericIdentifier(0)))
				assert.NotEmpty(t, gotRes.Data)
			}
		})
	}
}

func TestServer_ListTransactionsForHeight(t *testing.T) {
	tests := []struct {
		name string

		reqHeight uint64

		mockTransactions []flow.Identifier
		mockErr          error

		wantTransactions []flow.Identifier

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqHeight: mocks.GenericHeight,

			mockTransactions: mocks.GenericIdentifiers(5),

			wantTransactions: mocks.GenericIdentifiers(5),
			checkErr:         assert.NoError,
		},
		{
			name: "handles index failure",

			reqHeight: mocks.GenericHeight,

			mockErr: mocks.GenericError,

			checkErr: assert.Error,
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

			s := Server{index: index}

			req := &ListTransactionsForHeightRequest{
				Height: mocks.GenericHeight,
			}
			gotRes, gotErr := s.ListTransactionsForHeight(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.Height, mocks.GenericHeight)
				assert.Len(t, gotRes.TransactionIDs, 5)
			}
		})
	}
}

func TestServer_GetSeal(t *testing.T) {
	tests := []struct {
		name string

		reqSealID flow.Identifier

		mockSeal *flow.Seal
		mockErr  error

		wantSeal *flow.Seal

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqSealID: mocks.GenericIdentifier(0),

			mockSeal: mocks.GenericSeal(0),

			wantSeal: mocks.GenericSeal(0),
			checkErr: assert.NoError,
		},
		{
			name: "handles index failure",

			reqSealID: mocks.GenericIdentifier(0),
			mockErr:   mocks.GenericError,

			checkErr: assert.Error,
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
				codec: mocks.BaselineCodec(t),
				index: index,
			}

			req := GetSealRequest{
				SealID: mocks.ByteSlice(test.reqSealID),
			}
			gotRes, gotErr := s.GetSeal(context.Background(), &req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.SealID, mocks.ByteSlice(test.reqSealID))
				assert.NotEmpty(t, gotRes.Data)
			}
		})
	}
}

func TestServer_ListSealsForHeight(t *testing.T) {
	tests := []struct {
		name string

		reqHeight uint64

		mockSeals []flow.Identifier
		mockErr   error

		wantSeals []flow.Identifier

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqHeight: mocks.GenericHeight,

			mockSeals: mocks.GenericIdentifiers(5),

			wantSeals: mocks.GenericIdentifiers(5),
			checkErr:  assert.NoError,
		},
		{
			name: "handles index failure",

			reqHeight: mocks.GenericHeight,

			mockErr: mocks.GenericError,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := mocks.BaselineReader(t)
			index.SealsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
				return test.mockSeals, test.mockErr
			}

			s := Server{
				codec: mocks.BaselineCodec(t),
				index: index,
			}

			req := ListSealsForHeightRequest{
				Height: mocks.GenericHeight,
			}

			gotRes, gotErr := s.ListSealsForHeight(context.Background(), &req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.Height, test.reqHeight)
				assert.Len(t, gotRes.SealIDs, 5)
			}
		})
	}
}
