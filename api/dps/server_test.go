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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/models/convert"
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

func TestServer_GetFirst(t *testing.T) {

	testHeight := uint64(128)

	tests := []struct {
		name string

		mockHeight uint64
		mockErr    error

		wantRes *GetFirstResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			mockHeight: testHeight,
			mockErr:    nil,

			wantRes: &GetFirstResponse{
				Height: testHeight,
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			mockHeight: testHeight,
			mockErr:    errors.New("dummy error"),

			wantRes: nil,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.FirstFunc = func() (uint64, error) {
				return test.mockHeight, test.mockErr
			}

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

	var (
		testHeight = uint64(128)
	)

	tests := []struct {
		name string

		mockHeight uint64
		mockErr    error

		wantRes *GetLastResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			mockHeight: testHeight,
			mockErr:    nil,

			wantRes: &GetLastResponse{
				Height: testHeight,
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			mockHeight: testHeight,
			mockErr:    errors.New("dummy error"),

			wantRes: nil,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.LastFunc = func() (uint64, error) {
				return test.mockHeight, test.mockErr
			}

			req := &GetLastRequest{}

			gotRes, gotErr := s.GetLast(context.Background(), req)
			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetHeader(t *testing.T) {
	var (
		testHeight = uint64(128)
		testHeader = flow.Header{Height: testHeight}
		testData   = []byte(`testValue`)
	)

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

			reqHeight: testHeight,

			mockHeader: &testHeader,
			mockErr:    nil,

			wantHeight: testHeight,
			wantRes: &GetHeaderResponse{
				Height: testHeight,
				Data:   testData,
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			reqHeight: testHeight,

			mockHeader: &testHeader,
			mockErr:    errors.New("dummy error"),

			wantHeight: testHeight,
			wantRes:    nil,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			codec := &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, &flow.Header{}, v)
					return testData, nil
				},
			}
			index := &mocks.Reader{}
			s := Server{
				codec: codec,
				index: index,
			}

			var gotHeight uint64
			index.HeaderFunc = func(height uint64) (*flow.Header, error) {
				gotHeight = height
				return test.mockHeader, test.mockErr
			}

			req := &GetHeaderRequest{
				Height: test.reqHeight,
			}
			gotRes, gotErr := s.GetHeader(context.Background(), req)

			test.checkErr(t, gotErr)
			assert.Equal(t, test.wantHeight, gotHeight)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetCommit(t *testing.T) {

	var (
		testHeight = uint64(128)
		testCommit = flow.StateCommitment{0x1, 0x2}
	)

	tests := []struct {
		name string

		reqHeight uint64

		mockCommit flow.StateCommitment
		mockErr    error

		wantHeight uint64
		wantRes    *GetCommitResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqHeight: testHeight,

			mockCommit: testCommit,
			mockErr:    nil,

			wantHeight: testHeight,
			wantRes: &GetCommitResponse{
				Height: testHeight,
				Commit: testCommit[:],
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			reqHeight: testHeight,

			mockCommit: flow.StateCommitment{},
			mockErr:    errors.New("dummy error"),

			wantHeight: testHeight,
			wantRes:    nil,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			var gotHeight uint64
			index.CommitFunc = func(height uint64) (flow.StateCommitment, error) {
				gotHeight = height
				return test.mockCommit, test.mockErr
			}

			req := &GetCommitRequest{
				Height: test.reqHeight,
			}
			gotRes, gotErr := s.GetCommit(context.Background(), req)

			test.checkErr(t, gotErr)
			assert.Equal(t, test.wantHeight, gotHeight)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetEvents(t *testing.T) {
	var (
		testHeight = uint64(128)
		testEvents = []flow.Event{
			{TransactionID: flow.Identifier{0x1, 0x2}},
			{TransactionID: flow.Identifier{0x3, 0x4}},
		}
		testData  = []byte(`testValue`)
		testTypes = []flow.EventType{
			"type1",
			"type2",
		}
	)

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

			reqHeight: testHeight,
			reqTypes:  testTypes,

			mockEvents: testEvents,
			mockErr:    nil,

			wantHeight: testHeight,
			wantTypes:  testTypes,
			wantRes: &GetEventsResponse{
				Height: testHeight,
				Types:  convert.TypesToStrings(testTypes),
				Data:   testData,
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			reqHeight: testHeight,
			reqTypes:  testTypes,

			mockEvents: testEvents,
			mockErr:    errors.New("dummy error"),

			wantHeight: testHeight,
			wantTypes:  testTypes,
			wantRes:    nil,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			codec := &mocks.Codec{
				MarshalFunc: func(v interface{}) ([]byte, error) {
					assert.IsType(t, []flow.Event{}, v)
					return testData, nil
				},
			}
			index := &mocks.Reader{}
			s := Server{
				codec: codec,
				index: index,
			}

			var gotHeight uint64
			var gotTypes []flow.EventType
			index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
				gotHeight = height
				gotTypes = types
				return test.mockEvents, test.mockErr
			}

			req := &GetEventsRequest{
				Height: test.reqHeight,
				Types:  convert.TypesToStrings(test.reqTypes),
			}
			gotRes, gotErr := s.GetEvents(context.Background(), req)

			test.checkErr(t, gotErr)
			assert.Equal(t, test.wantHeight, gotHeight)
			assert.Equal(t, test.wantTypes, gotTypes)
			if gotErr == nil {
				assert.Equal(t, test.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetRegisters(t *testing.T) {

	var (
		testHeight = uint64(128)
		testPaths  = []ledger.Path{
			{0x1, 0x2},
			{0x3, 0x4},
		}
		testValues = []ledger.Value{
			{0x5, 0x6},
			{0x7, 0x8},
		}
	)

	tests := []struct {
		name string

		reqHeight uint64
		reqPaths  []ledger.Path

		mockValues []ledger.Value
		mockErr    error

		wantHeight uint64
		wantPaths  []ledger.Path
		wantRes    *GetRegistersResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqHeight: testHeight,
			reqPaths:  testPaths,

			mockValues: testValues,
			mockErr:    nil,

			wantHeight: testHeight,
			wantPaths:  testPaths,
			wantRes: &GetRegistersResponse{
				Height: testHeight,
				Paths:  convert.PathsToBytes(testPaths),
				Values: convert.ValuesToBytes(testValues),
			},

			checkErr: assert.NoError,
		},
		{
			name: "error case",

			reqHeight: testHeight,
			reqPaths:  testPaths,

			mockValues: testValues,
			mockErr:    errors.New("dummy error"),

			wantHeight: testHeight,
			wantPaths:  testPaths,
			wantRes:    nil,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			var gotHeight uint64
			var gotPaths []ledger.Path
			index.RegistersFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
				gotHeight = height
				gotPaths = paths
				return test.mockValues, test.mockErr
			}

			req := &GetRegistersRequest{
				Height: test.reqHeight,
				Paths:  convert.PathsToBytes(test.reqPaths),
			}
			gotRes, gotErr := s.GetRegisters(context.Background(), req)

			test.checkErr(t, gotErr)
			assert.Equal(t, test.wantHeight, gotHeight)
			assert.Equal(t, test.wantPaths, gotPaths)
			if gotErr == nil {
				assert.Equal(t, test.wantRes.Height, gotRes.Height)
				assert.EqualValues(t, test.wantRes.Paths, gotRes.Paths)
				assert.EqualValues(t, test.wantRes.Values, gotRes.Values)
			}
		})
	}
}

func TestServer_GetHeight(t *testing.T) {

	var (
		testHeight     = uint64(128)
		testBlockID, _ = flow.HexStringToIdentifier("98827808c61af6b29c7f16071e69a9bbfba40d0f96b572ce23994b3aa605c7c2")
	)

	tests := []struct {
		name string

		reqBlockID flow.Identifier

		mockHeight uint64
		mockErr    error

		wantBlockID flow.Identifier
		wantHeight  uint64

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqBlockID: testBlockID,

			mockHeight: testHeight,
			mockErr:    nil,

			wantBlockID: testBlockID,
			wantHeight:  testHeight,

			checkErr: assert.NoError,
		},
		{
			name: "error handling",

			reqBlockID: testBlockID,

			mockErr: errors.New("dummy error"),

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.HeightFunc = func(blockID flow.Identifier) (uint64, error) {
				return test.mockHeight, test.mockErr
			}

			req := &GetHeightRequest{
				BlockID: testBlockID[:],
			}
			gotRes, gotErr := s.GetHeight(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, test.wantHeight, gotRes.Height)
				assert.Equal(t, test.wantBlockID[:], gotRes.BlockID)
			}
		})
	}
}

func TestServer_GetTransaction(t *testing.T) {

	testTransactionID := flow.Identifier{0x98, 0x82, 0x78, 0x08, 0xc6, 0x1a, 0xf6, 0xb2, 0x9c, 0x7f, 0x16, 0x07, 0x1e, 0x69, 0xa9, 0xbb, 0xfb, 0xa4, 0x0d, 0x0f, 0x96, 0xb5, 0x72, 0xce, 0x23, 0x99, 0x4b, 0x3a, 0xa6, 0x05, 0xc7, 0xc2}
	testTransaction := flow.Transaction{
		TransactionBody: flow.TransactionBody{
			ReferenceBlockID:   flow.Identifier{},
			Script:             nil,
			Arguments:          nil,
			GasLimit:           0,
			ProposalKey:        flow.ProposalKey{},
			Payer:              flow.Address{},
			Authorizers:        nil,
			PayloadSignatures:  nil,
			EnvelopeSignatures: nil,
		},
		Status:           0,
		Events:           nil,
		ComputationSpent: 0,
		StartState:       flow.StateCommitment{},
		EndState:         flow.StateCommitment{},
	}

	tests := []struct {
		name string

		reqTransactionID flow.Identifier

		mockTransaction flow.Transaction
		mockErr         error

		wantTransaction flow.Transaction

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqTransactionID: testTransactionID,

			mockTransaction: testTransaction,

			wantTransaction: testTransaction,
			checkErr:        assert.NoError,
		},
		{
			name: "handles index failure",

			reqTransactionID: testTransactionID,

			mockErr: mocks.DummyError,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.TransactionFunc = func(transactionID flow.Identifier) (*flow.Transaction, error) {
				return &test.mockTransaction, test.mockErr
			}

			req := &GetTransactionRequest{
				TransactionID: testTransactionID[:],
			}
			gotRes, gotErr := s.GetTransaction(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.TransactionID, testTransactionID[:])
				assert.NotEmpty(t, gotRes.Data)
			}
		})
	}
}

func TestServer_ListTransactionsForBlock(t *testing.T) {

	testBlockID := flow.Identifier{0x98, 0x82, 0x78, 0x08, 0xc6, 0x1a, 0xf6, 0xb2, 0x9c, 0x7f, 0x16, 0x07, 0x1e, 0x69, 0xa9, 0xbb, 0xfb, 0xa4, 0x0d, 0x0f, 0x96, 0xb5, 0x72, 0xce, 0x23, 0x99, 0x4b, 0x3a, 0xa6, 0x05, 0xc7, 0xc2}
	testTransactionID1 := flow.Identifier{0x2a, 0x33, 0x55, 0xc2, 0x59, 0x92, 0xb0, 0xfb, 0xfc, 0x9f, 0x17, 0xd2, 0x78, 0xd2, 0xe9, 0x32, 0xbd, 0xc1, 0x1a, 0xad, 0x63, 0x59, 0x2f, 0xd1, 0xf1, 0xe5, 0x75, 0x71, 0x88, 0xee, 0x47, 0xbc}
	testTransactionID2 := flow.Identifier{0xc9, 0xdc, 0x08, 0x94, 0xc7, 0xee, 0x97, 0x29, 0x95, 0xed, 0x97, 0xe9, 0x8b, 0x07, 0x57, 0xa6, 0x71, 0xde, 0x3a, 0x00, 0x2d, 0xd8, 0xf5, 0xc0, 0xde, 0xfe, 0xfa, 0xbd, 0x1e, 0x6d, 0x92, 0x3a}
	testTransactionID3 := flow.Identifier{0x11, 0xb0, 0xd9, 0xdf, 0xdc, 0x37, 0xe2, 0x0b, 0x71, 0xf4, 0x56, 0x76, 0x10, 0x67, 0x8c, 0xf7, 0xf6, 0xbb, 0xbf, 0xd4, 0xd7, 0x31, 0x6b, 0x2a, 0xa5, 0xe4, 0x9f, 0x35, 0xca, 0x6b, 0xd5, 0x29}
	testTransactions := []flow.Identifier{testTransactionID1, testTransactionID2, testTransactionID3}

	tests := []struct {
		name string

		reqBlockID flow.Identifier

		mockTransactions []flow.Identifier
		mockErr          error

		wantTransactions []flow.Identifier

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqBlockID: testBlockID,

			mockTransactions: testTransactions,

			wantTransactions: testTransactions,
			checkErr:         assert.NoError,
		},
		{
			name: "handles index failure",

			reqBlockID: testBlockID,

			mockErr: mocks.DummyError,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.TransactionsFunc = func(blockID flow.Identifier) ([]flow.Identifier, error) {
				return test.mockTransactions, test.mockErr
			}

			req := &ListTransactionsForBlockRequest{
				BlockID: testBlockID[:],
			}
			gotRes, gotErr := s.ListTransactionsForBlock(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.BlockID, testBlockID[:])
				assert.Len(t, gotRes.TransactionIDs, 3)
			}
		})
	}
}

func TestServer_ListTransactionsForCollection(t *testing.T) {
	testTransactionID := flow.Identifier{0xd4, 0x7b, 0x1b, 0xf7, 0xf3, 0x7e, 0x19, 0x2c, 0xf8, 0x3d, 0x2b, 0xee, 0x3f, 0x63, 0x32, 0xb0, 0xd9, 0xb1, 0x5c, 0xa, 0xa7, 0x66, 0xd, 0x1e, 0x53, 0x22, 0xea, 0x96, 0x46, 0x67, 0xb3, 0x33}
	testCollectionID := flow.Identifier{0x98, 0x82, 0x78, 0x08, 0xc6, 0x1a, 0xf6, 0xb2, 0x9c, 0x7f, 0x16, 0x07, 0x1e, 0x69, 0xa9, 0xbb, 0xfb, 0xa4, 0x0d, 0x0f, 0x96, 0xb5, 0x72, 0xce, 0x23, 0x99, 0x4b, 0x3a, 0xa6, 0x05, 0xc7, 0xc2}
	testCollection := flow.LightCollection{Transactions: []flow.Identifier{testTransactionID, testTransactionID, testTransactionID, testTransactionID, testTransactionID}}

	tests := []struct {
		name string

		reqCollectionID flow.Identifier

		mockCollection flow.LightCollection
		mockErr        error

		wantCollection flow.LightCollection

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqCollectionID: testCollectionID,

			mockCollection: testCollection,

			wantCollection: testCollection,
			checkErr:       assert.NoError,
		},
		{
			name: "handles index failure",

			reqCollectionID: testCollectionID,

			mockErr: mocks.DummyError,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.CollectionFunc = func(transactionID flow.Identifier) (*flow.LightCollection, error) {
				return &test.mockCollection, test.mockErr
			}

			req := &ListTransactionsForCollectionRequest{
				CollectionID: testCollectionID[:],
			}
			gotRes, gotErr := s.ListTransactionsForCollection(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.CollectionID, testCollectionID[:])
				assert.Len(t, gotRes.TransactionIDs, 5)
			}
		})
	}
}

func TestServer_ListCollectionsForBlock(t *testing.T) {

	testBlockID := flow.Identifier{0x98, 0x82, 0x78, 0x08, 0xc6, 0x1a, 0xf6, 0xb2, 0x9c, 0x7f, 0x16, 0x07, 0x1e, 0x69, 0xa9, 0xbb, 0xfb, 0xa4, 0x0d, 0x0f, 0x96, 0xb5, 0x72, 0xce, 0x23, 0x99, 0x4b, 0x3a, 0xa6, 0x05, 0xc7, 0xc2}
	testCollectionID1 := flow.Identifier{0x2a, 0x33, 0x55, 0xc2, 0x59, 0x92, 0xb0, 0xfb, 0xfc, 0x9f, 0x17, 0xd2, 0x78, 0xd2, 0xe9, 0x32, 0xbd, 0xc1, 0x1a, 0xad, 0x63, 0x59, 0x2f, 0xd1, 0xf1, 0xe5, 0x75, 0x71, 0x88, 0xee, 0x47, 0xbc}
	testCollectionID2 := flow.Identifier{0xc9, 0xdc, 0x08, 0x94, 0xc7, 0xee, 0x97, 0x29, 0x95, 0xed, 0x97, 0xe9, 0x8b, 0x07, 0x57, 0xa6, 0x71, 0xde, 0x3a, 0x00, 0x2d, 0xd8, 0xf5, 0xc0, 0xde, 0xfe, 0xfa, 0xbd, 0x1e, 0x6d, 0x92, 0x3a}
	testCollectionID3 := flow.Identifier{0x11, 0xb0, 0xd9, 0xdf, 0xdc, 0x37, 0xe2, 0x0b, 0x71, 0xf4, 0x56, 0x76, 0x10, 0x67, 0x8c, 0xf7, 0xf6, 0xbb, 0xbf, 0xd4, 0xd7, 0x31, 0x6b, 0x2a, 0xa5, 0xe4, 0x9f, 0x35, 0xca, 0x6b, 0xd5, 0x29}
	testCollections := []flow.Identifier{testCollectionID1, testCollectionID2, testCollectionID3}

	tests := []struct {
		name string

		reqBlockID flow.Identifier

		mockCollections []flow.Identifier
		mockErr         error

		wantCollections []flow.Identifier

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name: "happy case",

			reqBlockID: testBlockID,

			mockCollections: testCollections,

			wantCollections: testCollections,
			checkErr:        assert.NoError,
		},
		{
			name: "handles index failure",

			reqBlockID: testBlockID,

			mockErr: mocks.DummyError,

			checkErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.CollectionsFunc = func(blockID flow.Identifier) ([]flow.Identifier, error) {
				return test.mockCollections, test.mockErr
			}

			req := &ListCollectionsForBlockRequest{
				BlockID: testBlockID[:],
			}
			gotRes, gotErr := s.ListCollectionsForBlock(context.Background(), req)

			test.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, gotRes.BlockID, testBlockID[:])
				assert.Len(t, gotRes.CollectionIDs, 3)
			}
		})
	}
}
