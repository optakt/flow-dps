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
	"errors"
	"testing"

	"github.com/onflow/flow-go/ledger"
	"github.com/stretchr/testify/assert"

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

		testPaths  = []ledger.Path{{0x1, 0x2}, {0x3, 0x4}}
		testValues = []ledger.Value{{0x5, 0x6}, {0x7, 0x8}}
	)

	tests := []struct {
		desc string

		reqHeight *uint64
		reqPaths  []ledger.Path

		mockValues []ledger.Value
		mockErr    error

		wantValues []ledger.Value
		wantHeight uint64
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			desc: "nominal case, height given",

			reqHeight: &testHeight,
			reqPaths:  testPaths,

			mockValues: testValues,
			mockErr:    nil,

			wantValues: testValues,
			wantHeight: testHeight,
			wantErr:    assert.NoError,
		},
		{
			desc: "nominal case, no height given",

			reqHeight: nil,
			reqPaths:  testPaths,

			mockValues: testValues,
			mockErr:    nil,

			wantValues: testValues,
			wantHeight: lastHeight,
			wantErr:    assert.NoError,
		},
		{
			desc: "state error",

			reqHeight: &testHeight,
			reqPaths:  testPaths,

			mockValues: testValues,
			mockErr:    errors.New("dummy error"),

			wantValues: nil,
			wantHeight: 0,
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
			for i, path := range test.reqPaths {
				m.RawState.On("Get", path).Return(test.mockValues[i], test.mockErr)
			}

			c := &Controller{
				state: m,
			}

			gotValues, gotHeight, err := c.ReadRegisters(test.reqHeight, test.reqPaths)
			test.wantErr(t, err)

			if err == nil {
				assert.Equal(t, test.wantValues, gotValues)
				assert.Equal(t, test.wantHeight, gotHeight)
			}

			m.AssertExpectations(t)
		})
	}
}
