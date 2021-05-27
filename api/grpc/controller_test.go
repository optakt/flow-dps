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

package grpc

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/stretchr/testify/assert"
)

func TestNewController(t *testing.T) {
	c := NewController(nil)
	assert.NotNil(t, c)
}

func TestController_GetRegister(t *testing.T) {
	var (
		testHeight uint64 = 312
		lastHeight uint64 = 835

		testKey   = []byte(`testKey`)
		testValue = []byte(`testValue`)
	)

	tests := []struct {
		desc string

		reqHeight *uint64
		reqKey    []byte

		stateGet func([]byte) ([]byte, error)

		wantResp *GetRegisterResponse
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			desc: "nominal case, height given",

			reqHeight: &testHeight,
			reqKey:    testKey,

			stateGet: func(key []byte) ([]byte, error) {
				assert.Equal(t, testKey, key)
				return testValue, nil
			},

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

			stateGet: func(key []byte) ([]byte, error) {
				assert.Equal(t, testKey, key)
				return testValue, nil
			},

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

			stateGet: func(key []byte) ([]byte, error) {
				return nil, errors.New("dummy error")
			},

			wantResp: nil,
			wantErr:  assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			mock := &stateMock{
				last: lastMock{
					height: func() uint64 {
						return lastHeight
					},
				},
				withHeight: func(_ uint64) dps.Raw {
					return stateMock{
						get: test.stateGet,
					}
				},
			}

			c := &Controller{
				state: mock,
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
		})
	}
}

func TestController_GetValues(t *testing.T) {
	var testVersion uint64 = 42
	var (
		testKeys = []*Key{
			{
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
			},
		}
		testValue     = []byte(`testValue`)
		testCommit    = flow.StateCommitment{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
		testCommitHex = hex.EncodeToString(testCommit[:])
		lastCommit    = flow.StateCommitment{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2}
		lastCommitHex = hex.EncodeToString(lastCommit[:])
	)

	tests := []struct {
		desc string

		reqCommit  []byte
		reqVersion *uint64
		reqKeys    []*Key

		stateGet func(*ledger.Query) ([]ledger.Value, error)

		wantResp *GetValuesResponse
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			desc: "nominal case, version and commit hash given",

			reqKeys:    testKeys,
			reqCommit:  testCommit[:],
			reqVersion: &testVersion,

			stateGet: func(query *ledger.Query) ([]ledger.Value, error) {
				assert.Equal(t, testCommitHex[:], query.State().String())
				return []ledger.Value{
					testValue,
				}, nil
			},

			wantResp: &GetValuesResponse{
				Values: [][]byte{testValue},
			},
			wantErr: assert.NoError,
		},
		{
			desc: "nominal case, version given, using latest commit",

			reqKeys:    testKeys,
			reqVersion: &testVersion,

			stateGet: func(query *ledger.Query) ([]ledger.Value, error) {
				assert.Equal(t, lastCommitHex[:], query.State().String())
				return []ledger.Value{
					testValue,
				}, nil
			},

			wantResp: &GetValuesResponse{
				Values: [][]byte{testValue},
			},
			wantErr: assert.NoError,
		},
		{
			desc: "nominal case, no version or commit hash given",

			reqKeys: testKeys,

			stateGet: func(query *ledger.Query) ([]ledger.Value, error) {
				assert.Equal(t, lastCommitHex[:], query.State().String())
				return []ledger.Value{
					testValue,
				}, nil
			},

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

			stateGet: func(_ *ledger.Query) ([]ledger.Value, error) {
				return nil, errors.New("dummy error")
			},

			wantErr: assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			mock := &stateMock{
				last: lastMock{
					commit: func() flow.StateCommitment {
						return lastCommit
					},
				},
				ledger: ledgerMock{
					withVersion: func(version uint8) dps.Ledger {
						if test.reqVersion == nil {
							t.Fail()
						} else {
							assert.Equal(t, uint8(*test.reqVersion), version)
						}

						return &ledgerMock{
							get: test.stateGet,
						}
					},
					get: test.stateGet,
				},
			}

			c := &Controller{
				state: mock,
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
		})
	}
}

type stateMock struct {
	last       lastMock
	ledger     ledgerMock
	get        func([]byte) ([]byte, error)
	withHeight func(uint64) dps.Raw
}

func (s stateMock) Get(key []byte) ([]byte, error) {
	return s.get(key)
}

func (s stateMock) Last() dps.Last {
	return s.last
}

func (s stateMock) Index() dps.Index {
	panic("implement me")
}

func (s stateMock) Chain() dps.Chain {
	panic("implement me")
}

func (s stateMock) Height() dps.Height {
	panic("implement me")
}

func (s stateMock) Commit() dps.Commit {
	panic("implement me")
}

func (s stateMock) Raw() dps.Raw {
	return s
}

func (s stateMock) WithHeight(height uint64) dps.Raw {
	return s.withHeight(height)
}

func (s stateMock) Ledger() dps.Ledger {
	return s.ledger
}

type lastMock struct {
	height func() uint64
	commit func() flow.StateCommitment
}

func (l lastMock) Height() uint64 {
	return l.height()
}

func (l lastMock) Commit() flow.StateCommitment {
	return l.commit()
}

type ledgerMock struct {
	get         func(*ledger.Query) ([]ledger.Value, error)
	withVersion func(uint8) dps.Ledger
}

func (l ledgerMock) WithVersion(version uint8) dps.Ledger {
	return l.withVersion(version)
}

func (l ledgerMock) Get(query *ledger.Query) ([]ledger.Value, error) {
	return l.get(query)
}
