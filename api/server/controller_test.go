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

package server

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewController(t *testing.T) {
	m := mocks.NewState()

	c := NewController(m)
	assert.NotNil(t, c)
	assert.Equal(t, m, c.state)
}

func TestController_GetRegister(t *testing.T) {
	var (
		testHeight uint64 = 128
		lastHeight uint64 = 256

		testKey   = []byte(`testKey`)
		testValue = []byte(`testValue`)
	)

	tests := []struct {
		desc string

		reqHeight *uint64
		reqKey    []byte

		mockValue []byte
		mockErr   error

		wantHeight uint64
		wantResp   *GetRegisterResponse
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			desc: "nominal case, height given",

			reqHeight: &testHeight,
			reqKey:    testKey,

			mockValue: testValue,

			wantHeight: testHeight,
			wantResp: &GetRegisterResponse{
				Height: testHeight,
				Key:    testKey,
				Value:  testValue,
			},
			wantErr: assert.NoError,
		},
		{
			desc: "nominal case, no height given",

			reqKey: testKey,

			mockValue: testValue,

			wantHeight: lastHeight,
			wantResp: &GetRegisterResponse{
				Height: lastHeight,
				Key:    testKey,
				Value:  testValue,
			},
			wantErr: assert.NoError,
		},
		{
			desc: "state error",

			reqKey: testKey,

			mockErr: errors.New("dummy error"),

			wantHeight: lastHeight,
			wantResp:   nil,
			wantErr:    assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			m := mocks.NewState()
			m.LastState.On("Height").Return(lastHeight).Once()
			m.RawState.On("WithHeight", test.wantHeight).Return(m.RawState).Once()
			m.RawState.On("Get", test.reqKey).Return(test.mockValue, test.mockErr)

			c := &Controller{
				state: m,
			}

			req := &GetRegisterRequest{
				Height: test.reqHeight,
				Key:    test.reqKey,
			}

			got, err := c.GetRegister(context.Background(), req)
			test.wantErr(t, err)

			if test.wantResp != nil {
				assert.Equal(t, test.wantResp, got)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestController_GetValues(t *testing.T) {
	var (
		testKey = &Key{
			Parts: []*KeyPart{
				{
					Type:  0,
					Value: []byte(`testOwner`),
				},
				{
					Type:  1,
					Value: []byte(`testController`),
				},
				{
					Type:  2,
					Value: []byte(`testKey`),
				},
			},
		}
		testKeys    = []*Key{testKey}
		testVersion = uint64(42)
		testValue   = []byte(`testValue`)
		testValues  = []ledger.Value{ledger.Value(testValue)}
	)

	testCommit, err := flow.ToStateCommitment([]byte("0d339afb6de1aa21b7afbcef3278c8ee"))
	require.NoError(t, err)
	lastCommit, err := flow.ToStateCommitment([]byte("25026807db966e6464d17d99b780b9f3"))
	require.NoError(t, err)

	tests := []struct {
		desc string

		reqCommit  []byte
		reqVersion *uint64
		reqKeys    []*Key

		mockValues []ledger.Value
		mockErr    error

		wantResp *GetValuesResponse
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			desc: "nominal case, version and commit hash given",

			reqKeys:    testKeys,
			reqCommit:  testCommit[:],
			reqVersion: &testVersion,

			mockValues: testValues,

			wantResp: &GetValuesResponse{
				Values: [][]byte{testValue},
			},
			wantErr: assert.NoError,
		},
		{
			desc: "nominal case, version given, using latest commit",

			reqKeys:    testKeys,
			reqVersion: &testVersion,

			mockValues: testValues,

			wantResp: &GetValuesResponse{
				Values: [][]byte{testValue},
			},
			wantErr: assert.NoError,
		},
		{
			desc: "nominal case, no version or commit hash given",

			reqKeys: testKeys,

			mockValues: testValues,

			wantResp: &GetValuesResponse{
				Values: [][]byte{testValue},
			},
			wantErr: assert.NoError,
		},
		{
			desc: "nominal case, three keys given",

			reqKeys: []*Key{testKey, testKey, testKey},

			mockValues: testValues,

			wantResp: &GetValuesResponse{
				Values: [][]byte{testValue},
			},
			wantErr: assert.NoError,
		},
		{
			desc: "no keys given",

			reqKeys: []*Key{},

			mockValues: testValues,

			wantResp: &GetValuesResponse{
				Values: [][]byte{testValue},
			},
			wantErr: assert.NoError,
		},
		{
			desc: "invalid commit hash in request",

			reqKeys:   testKeys,
			reqCommit: []byte(`not a hexadecimal value`),

			wantErr: assert.Error,
		},
		{
			desc: "state get returns an error",

			reqKeys: testKeys,

			mockErr: errors.New("dummy error"),

			wantErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			m := mocks.NewState()
			m.LastState.On("Commit").Return(lastCommit).Once()
			m.LedgerState.On("Get", mock.Anything).Return(test.mockValues, test.mockErr).Once()
			if test.reqVersion != nil {
				m.LedgerState.On("WithVersion", uint8(*test.reqVersion)).Return(m.LedgerState).Once()
			}

			c := &Controller{
				state: m,
			}

			req := &GetValuesRequest{
				Keys:    test.reqKeys,
				Hash:    test.reqCommit,
				Version: test.reqVersion,
			}

			got, err := c.GetValues(context.Background(), req)
			test.wantErr(t, err)

			if test.wantResp != nil {
				assert.Equal(t, test.wantResp, got)
			}

			m.AssertExpectations(t)
		})
	}
}
