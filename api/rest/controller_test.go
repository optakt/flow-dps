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

package rest_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/assert"
	tmock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/rest"
	"github.com/optakt/flow-dps/models/dps/mock"
)

func TestController_GetRegister(t *testing.T) {
	var testKey = []byte{0x74, 0x65, 0x73, 0x74, 0x4b, 0x65, 0x79}
	const (
		keyHex            = "746573744b6579"
		testValue         = "testValue"
		valueHex          = "7465737456616c7565"
		testHeight uint64 = 425
		lastHeight uint64 = 835
	)

	tests := []struct {
		desc        string
		key         string
		heightParam string
		lastHeight  uint64

		mockValue []byte
		mockErr   error

		wantHeight   uint64
		wantStatus   int
		wantResponse *rest.RegisterResponse
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			desc:        "nominal case with heightParam",
			key:         keyHex,
			heightParam: fmt.Sprint(testHeight),
			lastHeight:  lastHeight,

			mockValue: []byte(testValue),

			wantHeight: testHeight,
			wantStatus: http.StatusOK,
			wantResponse: &rest.RegisterResponse{
				Height: testHeight,
				Key:    keyHex,
				Value:  valueHex,
			},
			wantErr: assert.NoError,
		},
		{
			desc:       "nominal case using last height",
			key:        keyHex,
			lastHeight: lastHeight,

			mockValue: []byte(testValue),

			wantHeight: lastHeight,
			wantStatus: http.StatusOK,
			wantResponse: &rest.RegisterResponse{
				Height: lastHeight,
				Key:    keyHex,
				Value:  valueHex,
			},
			wantErr: assert.NoError,
		},
		{
			desc: "invalid key in ctx parameters",
			key:  "not-hexadecimal",

			wantHeight: lastHeight,
			wantStatus: http.StatusBadRequest,
			wantErr:    assert.Error,
		},
		{
			desc:        "invalid heightParam (negative value)",
			key:         keyHex,
			heightParam: "not a number",

			wantHeight: lastHeight,
			wantStatus: http.StatusBadRequest,
			wantErr:    assert.Error,
		},
		{
			desc:       "key not found",
			key:        keyHex,
			lastHeight: lastHeight,

			mockErr: badger.ErrKeyNotFound,

			wantHeight: lastHeight,
			wantStatus: http.StatusNotFound,
			wantErr:    assert.Error,
		},
		{
			desc:       "internal state error",
			key:        keyHex,
			lastHeight: lastHeight,

			mockErr: errors.New("dummy error"),

			wantHeight: lastHeight,
			wantStatus: http.StatusInternalServerError,
			wantErr:    assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			// Forge echo context and insert parameters.
			e := echo.New()

			u, err := url.Parse("https://1.2.3.4/values/:keys")
			require.NoError(t, err)

			q := u.Query()
			if test.heightParam != "" {
				q.Set("height", test.heightParam)
			}
			u.RawQuery = q.Encode()

			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.SetPath(fmt.Sprintf("%s?%s", u.Path, u.RawQuery))
			ctx.SetParamNames("key")
			ctx.SetParamValues(test.key)

			// Create mock for state.
			m := mock.NewState()
			m.LastState.On("Height").Return(lastHeight).Once()
			m.RawState.On("WithHeight", test.wantHeight).Return(m.RawState).Once()
			m.RawState.On("Get", testKey).Return(test.mockValue, test.mockErr)

			// Create controller and begin test.
			c := rest.NewController(m)

			err = c.GetRegister(ctx)
			test.wantErr(t, err)

			if test.wantStatus != http.StatusOK {
				// Note: See https://github.com/labstack/echo/issues/593
				httpErr, ok := err.(*echo.HTTPError)
				require.True(t, ok)
				assert.Equal(t, test.wantStatus, httpErr.Code)
			} else {
				assert.Equal(t, test.wantStatus, rec.Code)
			}

			if test.wantResponse != nil {
				b, err := io.ReadAll(rec.Result().Body)
				require.NoError(t, err)

				var gotResponse rest.RegisterResponse
				err = json.Unmarshal(b, &gotResponse)
				require.NoError(t, err)

				assert.Equal(t, test.wantResponse, &gotResponse)
			}
		})
	}
}

func TestController_GetValue(t *testing.T) {
	const (
		validKeys         = "0.,1.,2.746573744b6579:0.,1.,2.746573744b657932"
		invalidKeys       = "non-hexadecimal value"
		customCommitParam = "3732396638393566663833126661376434313137623730333731346338623337"
		value             = "testValue"
		valueHex          = "7465737456616c7565"
	)
	var testValues = []ledger.Value{ledger.Value(value)}
	var lastCommit = flow.StateCommitment{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2}

	tests := []struct {
		desc string

		keys         string
		versionParam string
		commitParam  string

		mockValues []ledger.Value
		mockErr    error

		wantStatus   int
		wantResponse []string
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			desc:         "nominal case with commitParam",
			keys:         validKeys,
			versionParam: "47",
			commitParam:  customCommitParam,

			mockValues: testValues,

			wantStatus: http.StatusOK,
			wantResponse: []string{
				valueHex,
			},
			wantErr: assert.NoError,
		},
		{
			desc:         "nominal case without commitParam",
			keys:         validKeys,
			versionParam: "47",

			mockValues: testValues,

			wantStatus: http.StatusOK,
			wantResponse: []string{
				valueHex,
			},
			wantErr: assert.NoError,
		},
		{
			desc: "invalid keys",
			keys: invalidKeys,

			wantStatus: http.StatusBadRequest,
			wantErr:    assert.Error,
		},
		{
			desc:        "invalid commit param",
			keys:        validKeys,
			commitParam: "non-hexadecimal value",

			wantStatus: http.StatusBadRequest,
			wantErr:    assert.Error,
		},
		{
			desc:         "invalid version param",
			keys:         validKeys,
			versionParam: "not a number",

			wantStatus: http.StatusBadRequest,
			wantErr:    assert.Error,
		},
		{
			desc: "key/commit not found",
			keys: validKeys,

			mockErr: badger.ErrKeyNotFound,

			wantStatus: http.StatusNotFound,
			wantErr:    assert.Error,
		},
		{
			desc: "internal state error",
			keys: validKeys,

			mockErr: errors.New("dummy error"),

			wantStatus: http.StatusInternalServerError,
			wantErr:    assert.Error,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			// Forge echo context and insert parameters.
			e := echo.New()

			u, err := url.Parse("https://1.2.3.4/values/:keys")
			require.NoError(t, err)

			q := u.Query()
			if test.commitParam != "" {
				q.Set("hash", test.commitParam)
			}
			if test.versionParam != "" {
				q.Set("version", test.versionParam)
			}
			u.RawQuery = q.Encode()

			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)
			ctx.SetPath(fmt.Sprintf("%s?%s", u.Path, u.RawQuery))
			ctx.SetParamNames("keys")
			ctx.SetParamValues(test.keys)

			// Create mock for state.
			m := mock.NewState()
			m.LastState.On("Commit").Return(lastCommit).Once()
			m.LedgerState.On("Get", tmock.Anything).Return(test.mockValues, test.mockErr).Once()
			if test.versionParam != "" {
				m.LedgerState.On("WithVersion", tmock.Anything).Return(m.LedgerState).Once()
			}

			// Create controller and begin test.
			c := rest.NewController(m)

			err = c.GetValue(ctx)
			test.wantErr(t, err)

			if test.wantStatus != http.StatusOK {
				// Note: See https://github.com/labstack/echo/issues/593
				httpErr, ok := err.(*echo.HTTPError)
				require.True(t, ok)
				assert.Equal(t, test.wantStatus, httpErr.Code)
			} else {
				assert.Equal(t, test.wantStatus, rec.Code)
			}

			if test.wantResponse != nil {
				b, err := io.ReadAll(rec.Result().Body)
				require.NoError(t, err)

				var gotResponse []string
				err = json.Unmarshal(b, &gotResponse)
				require.NoError(t, err)

				assert.Equal(t, test.wantResponse, gotResponse)
			}
		})
	}
}
