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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewServer(t *testing.T) {

	index := &mocks.Reader{}
	s := NewServer(index)

	assert.NotNil(t, s)
	assert.NotNil(t, s.codec)
	assert.Equal(t, index, s.index)
}

func TestServer_GetFirst(t *testing.T) {

	var (
		testHeight = uint64(128)
	)

	vectors := []struct {
		description string

		mockHeight uint64
		mockErr    error

		wantRes *GetFirstResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			description: "happy case",

			mockHeight: testHeight,
			mockErr:    nil,

			wantRes: &GetFirstResponse{
				Height: testHeight,
			},

			checkErr: assert.NoError,
		},
		{
			description: "error case",

			mockHeight: testHeight,
			mockErr:    errors.New("dummy error"),

			wantRes: nil,

			checkErr: assert.Error,
		},
	}

	for _, vector := range vectors {
		vector := vector
		t.Run(vector.description, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.FirstFunc = func() (uint64, error) {
				return vector.mockHeight, vector.mockErr
			}

			req := &GetFirstRequest{}

			gotRes, gotErr := s.GetFirst(context.Background(), req)
			vector.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, vector.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetLast(t *testing.T) {

	var (
		testHeight = uint64(128)
	)

	vectors := []struct {
		description string

		mockHeight uint64
		mockErr    error

		wantRes *GetLastResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			description: "happy case",

			mockHeight: testHeight,
			mockErr:    nil,

			wantRes: &GetLastResponse{
				Height: testHeight,
			},

			checkErr: assert.NoError,
		},
		{
			description: "error case",

			mockHeight: testHeight,
			mockErr:    errors.New("dummy error"),

			wantRes: nil,

			checkErr: assert.Error,
		},
	}

	for _, vector := range vectors {
		vector := vector
		t.Run(vector.description, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.LastFunc = func() (uint64, error) {
				return vector.mockHeight, vector.mockErr
			}

			req := &GetLastRequest{}

			gotRes, gotErr := s.GetLast(context.Background(), req)
			vector.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, vector.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetHeader(t *testing.T) {

	var (
		testCodec, _ = dps.Encoding.EncMode()
		testHeight   = uint64(128)
		testHeader   = flow.Header{Height: testHeight}
		testData, _  = testCodec.Marshal(testHeader)
	)

	vectors := []struct {
		description string

		reqHeight uint64

		mockHeader *flow.Header
		mockErr    error

		wantHeight uint64
		wantRes    *GetHeaderResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			description: "happy case",

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
			description: "error case",

			reqHeight: testHeight,

			mockHeader: &testHeader,
			mockErr:    errors.New("dummy error"),

			wantHeight: testHeight,
			wantRes:    nil,

			checkErr: assert.Error,
		},
	}

	for _, vector := range vectors {
		vector := vector
		t.Run(vector.description, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{
				codec: testCodec,
				index: index,
			}

			var gotHeight uint64
			index.HeaderFunc = func(height uint64) (*flow.Header, error) {
				gotHeight = height
				return vector.mockHeader, vector.mockErr
			}

			req := &GetHeaderRequest{
				Height: vector.reqHeight,
			}
			gotRes, gotErr := s.GetHeader(context.Background(), req)

			vector.checkErr(t, gotErr)
			assert.Equal(t, vector.wantHeight, gotHeight)
			if gotErr == nil {
				assert.Equal(t, vector.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetCommit(t *testing.T) {

	var (
		testHeight = uint64(128)
		testCommit = flow.StateCommitment{0x1, 0x2}
	)

	vectors := []struct {
		description string

		reqHeight uint64

		mockCommit flow.StateCommitment
		mockErr    error

		wantHeight uint64
		wantRes    *GetCommitResponse

		checkErr assert.ErrorAssertionFunc
	}{
		{
			description: "happy case",

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
			description: "error case",

			reqHeight: testHeight,

			mockCommit: flow.StateCommitment{},
			mockErr:    errors.New("dummy error"),

			wantHeight: testHeight,
			wantRes:    nil,

			checkErr: assert.Error,
		},
	}

	for _, vector := range vectors {
		vector := vector
		t.Run(vector.description, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			var gotHeight uint64
			index.CommitFunc = func(height uint64) (flow.StateCommitment, error) {
				gotHeight = height
				return vector.mockCommit, vector.mockErr
			}

			req := &GetCommitRequest{
				Height: vector.reqHeight,
			}
			gotRes, gotErr := s.GetCommit(context.Background(), req)

			vector.checkErr(t, gotErr)
			assert.Equal(t, vector.wantHeight, gotHeight)
			if gotErr == nil {
				assert.Equal(t, vector.wantRes, gotRes)
			}
		})
	}
}

func TestServer_GetEvents(t *testing.T) {

	var (
		testCodec, _ = dps.Encoding.EncMode()
		testHeight   = uint64(128)
		testEvents   = []flow.Event{
			{TransactionID: flow.Identifier{0x1, 0x2}},
			{TransactionID: flow.Identifier{0x3, 0x4}},
		}
		testData, _ = testCodec.Marshal(testEvents)
		testTypes   = []flow.EventType{
			"type1",
			"type2",
		}
	)

	vectors := []struct {
		description string

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
			description: "happy case",

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
			description: "error case",

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

	for _, vector := range vectors {
		vector := vector
		t.Run(vector.description, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{
				codec: testCodec,
				index: index,
			}

			var gotHeight uint64
			var gotTypes []flow.EventType
			index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
				gotHeight = height
				gotTypes = types
				return vector.mockEvents, vector.mockErr
			}

			req := &GetEventsRequest{
				Height: vector.reqHeight,
				Types:  convert.TypesToStrings(vector.reqTypes),
			}
			gotRes, gotErr := s.GetEvents(context.Background(), req)

			vector.checkErr(t, gotErr)
			assert.Equal(t, vector.wantHeight, gotHeight)
			assert.Equal(t, vector.wantTypes, gotTypes)
			if gotErr == nil {
				assert.Equal(t, vector.wantRes, gotRes)
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

	vectors := []struct {
		description string

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
			description: "happy case",

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
			description: "error case",

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

	for _, vector := range vectors {
		vector := vector
		t.Run(vector.description, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			var gotHeight uint64
			var gotPaths []ledger.Path
			index.RegistersFunc = func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
				gotHeight = height
				gotPaths = paths
				return vector.mockValues, vector.mockErr
			}

			req := &GetRegistersRequest{
				Height: vector.reqHeight,
				Paths:  convert.PathsToBytes(vector.reqPaths),
			}
			gotRes, gotErr := s.GetRegisters(context.Background(), req)

			vector.checkErr(t, gotErr)
			assert.Equal(t, vector.wantHeight, gotHeight)
			assert.Equal(t, vector.wantPaths, gotPaths)
			if gotErr == nil {
				assert.Equal(t, vector.wantRes.Height, gotRes.Height)
				assert.EqualValues(t, vector.wantRes.Paths, gotRes.Paths)
				assert.EqualValues(t, vector.wantRes.Values, gotRes.Values)
			}
		})
	}
}

func TestServer_GetHeight(t *testing.T) {

	var (
		testHeight     = uint64(128)
		testBlockID, _ = flow.HexStringToIdentifier("98827808c61af6b29c7f16071e69a9bbfba40d0f96b572ce23994b3aa605c7c2")
	)

	vectors := []struct {
		description string

		reqBlockID flow.Identifier

		mockHeight uint64
		mockErr    error

		wantBlockID flow.Identifier
		wantHeight  uint64

		checkErr assert.ErrorAssertionFunc
	}{
		{
			description: "happy case",

			reqBlockID: testBlockID,

			mockHeight: testHeight,
			mockErr:    nil,

			wantBlockID: testBlockID,
			wantHeight:  testHeight,

			checkErr: assert.NoError,
		},
		{
			description: "error handling",

			reqBlockID: testBlockID,

			mockErr: errors.New("dummy error"),

			checkErr: assert.Error,
		},
	}

	for _, vector := range vectors {
		vector := vector
		t.Run(vector.description, func(t *testing.T) {
			t.Parallel()

			index := &mocks.Reader{}
			s := Server{index: index}

			index.HeightFunc = func(blockID flow.Identifier) (uint64, error) {
				return vector.mockHeight, vector.mockErr
			}

			req := &GetHeightRequest{
				BlockID: testBlockID[:],
			}
			gotRes, gotErr := s.GetHeight(context.Background(), req)

			vector.checkErr(t, gotErr)
			if gotErr == nil {
				assert.Equal(t, vector.wantHeight, gotRes.Height)
				assert.Equal(t, vector.wantBlockID[:], gotRes.BlockID)
			}
		})
	}
}
