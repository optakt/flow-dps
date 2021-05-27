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
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/rest"
	"github.com/optakt/flow-dps/models/dps"
)

func TestController_GetRegister(t *testing.T) {
	const (
		keyHex     = "746573744b6579"
		value      = "testValue"
		valueHex   = "7465737456616c7565"
		lastHeight = 835
	)

	tests := []struct {
		desc        string
		key         string
		heightParam string
		lastHeight  uint64

		stateGet func([]byte) ([]byte, error)

		wantStatus   int
		wantResponse *rest.RegisterResponse
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			desc:        "nominal case with heightParam",
			key:         keyHex,
			heightParam: "425",
			lastHeight:  lastHeight,

			stateGet: func(bytes []byte) ([]byte, error) {
				return []byte(value), nil
			},

			wantStatus: http.StatusOK,
			wantResponse: &rest.RegisterResponse{
				Height: 425,
				Key:    keyHex,
				Value:  valueHex,
			},
			wantErr: assert.NoError,
		},
		{
			desc:       "nominal case using last height",
			key:        keyHex,
			lastHeight: lastHeight,

			stateGet: func(bytes []byte) ([]byte, error) {
				return []byte(value), nil
			},

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

			wantStatus: http.StatusBadRequest,
			wantErr:    assert.Error,
		},
		{
			desc:        "invalid heightParam (negative value)",
			key:         keyHex,
			heightParam: "not a number",

			wantStatus: http.StatusBadRequest,
			wantErr:    assert.Error,
		},
		{
			desc:       "key not found",
			key:        keyHex,
			lastHeight: lastHeight,

			stateGet: func(bytes []byte) ([]byte, error) {
				return nil, badger.ErrKeyNotFound
			},

			wantStatus: http.StatusNotFound,
			wantErr:    assert.Error,
		},
		{
			desc:       "internal state error",
			key:        keyHex,
			lastHeight: lastHeight,

			stateGet: func(bytes []byte) ([]byte, error) {
				return nil, errors.New("dummy error")
			},

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

			u, err := url.Parse(fmt.Sprintf("https://1.2.3.4/values/:keys"))
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
			stateMock := &stateMock{
				withHeight: func(height uint64) dps.Raw {
					return &stateMock{
						get: test.stateGet,
					}
				},
				last: lastMock{
					height: func() uint64 {
						return test.lastHeight
					},
				},
			}

			// Create controller and begin test.
			c := rest.NewController(stateMock)

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
		lastCommitHex     = "0102030405060708090a0102030405060708090a0102030405060708090a0102"
		customCommitParam = "3732396638393566663833126661376434313137623730333731346338623337"
		value             = "testValue"
		valueHex          = "7465737456616c7565"
	)

	var lastCommit = flow.StateCommitment{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1, 2}

	tests := []struct {
		desc string

		keys         string
		versionParam string
		commitParam  string

		stateGet func(*ledger.Query) ([]ledger.Value, error)

		wantStatus   int
		wantResponse []string
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			desc:         "nominal case with commitParam",
			keys:         validKeys,
			versionParam: "47",
			commitParam:  customCommitParam,

			stateGet: func(query *ledger.Query) ([]ledger.Value, error) {
				assert.Equal(t, customCommitParam, query.State().String())
				return []ledger.Value{
					ledger.Value(value),
				}, nil
			},

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

			stateGet: func(query *ledger.Query) ([]ledger.Value, error) {
				assert.Equal(t, lastCommitHex, query.State().String())
				return []ledger.Value{
					ledger.Value(value),
				}, nil
			},

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

			stateGet: func(query *ledger.Query) ([]ledger.Value, error) {
				return nil, badger.ErrKeyNotFound
			},

			wantStatus: http.StatusNotFound,
			wantErr:    assert.Error,
		},
		{
			desc: "internal state error",
			keys: validKeys,

			stateGet: func(query *ledger.Query) ([]ledger.Value, error) {
				return nil, errors.New("dummy error")
			},

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

			u, err := url.Parse(fmt.Sprintf("https://1.2.3.4/values/:keys"))
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
			stateMock := &stateMock{
				last: lastMock{
					commit: func() flow.StateCommitment {
						return flow.StateCommitment(lastCommit)
					},
				},
				ledger: ledgerMock{
					withVersion: func(version uint8) dps.Ledger {
						assert.Equal(t, test.versionParam, fmt.Sprint(version))
						return &ledgerMock{
							get: test.stateGet,
						}
					},
					get: test.stateGet,
				},
			}

			// Create controller and begin test.
			c := rest.NewController(stateMock)

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
