// +build integration

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

package rosetta_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/models/identifier"
	"github.com/optakt/flow-dps/rosetta/invoker"
	"github.com/optakt/flow-dps/rosetta/retriever"
	"github.com/optakt/flow-dps/rosetta/scripts"
	"github.com/optakt/flow-dps/rosetta/validator"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/testing/snapshots"
)

func setupDB(t *testing.T) *badger.DB {
	t.Helper()

	opts := badger.DefaultOptions("").
		WithInMemory(true).
		WithReadOnly(true).
		WithLogger(nil)

	db, err := badger.Open(opts)
	require.NoError(t, err)

	reader := hex.NewDecoder(strings.NewReader(snapshots.Rosetta))

	err = db.Load(reader, runtime.GOMAXPROCS(0))
	require.NoError(t, err)

	return db
}

func setupAPI(t *testing.T, db *badger.DB) *rosetta.Data {
	t.Helper()

	index := index.NewReader(db)

	params := dps.FlowParams[dps.FlowTestnet]
	generator := scripts.NewGenerator(params)
	invoke := invoker.New(index)
	validate := validator.New(params)
	retrieve := retriever.New(generator, invoke)
	controller := rosetta.NewData(validate, retrieve)

	return controller
}
func TestGetBalance(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	tests := []struct {
		name string

		request rosetta.BalanceRequest

		wantStatusCode int
		wantBalance    string
		wantHandlerErr assert.ErrorAssertionFunc
	}{
		// TODO: use a different 'valid balance request' - one that has an actual, non-zero balance
		{
			name:           "valid balance request",
			request:        getBalanceRequest("8c5303eaa26202d6", 0, "d47b1bf7f37e192cf83d2bee3f6332b0d9b15c0aa7660d1e5322ea964667b333"),
			wantBalance:    "0",
			wantStatusCode: http.StatusOK,
			wantHandlerErr: assert.NoError,
		},
		{
			name:           "empty balance request",
			request:        rosetta.BalanceRequest{},
			wantStatusCode: http.StatusUnprocessableEntity,
			wantHandlerErr: assert.Error,
		},
		{
			name:           "block hash and height mismatch",
			request:        getBalanceRequest("8c5303eaa26202d6", 99, "d47b1bf7f37e192cf83d2bee3f6332b0d9b15c0aa7660d1e5322ea964667b333"),
			wantStatusCode: http.StatusUnprocessableEntity,
			wantHandlerErr: assert.Error,
		},
		{
			name:           "invalid account address",
			request:        getBalanceRequest("invalid_address", 0, "d47b1bf7f37e192cf83d2bee3f6332b0d9b15c0aa7660d1e5322ea964667b333"),
			wantStatusCode: http.StatusUnprocessableEntity,
			wantHandlerErr: assert.Error,
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			// prepare request payload (JSON)
			enc, err := json.Marshal(test.request)
			require.NoError(t, err)

			// create request
			req := httptest.NewRequest(http.MethodPost, "/account/balance", bytes.NewReader(enc))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			rec := httptest.NewRecorder()

			ctx := echo.New().NewContext(req, rec)

			// execute the request
			err = api.Balance(ctx)
			test.wantHandlerErr(t, err)

			// validate response data

			// validate status code
			if test.wantStatusCode != http.StatusOK {

				// ugly workaround for HTTP status code set by router vs set on context
				e, ok := err.(*echo.HTTPError)
				require.True(t, ok)
				assert.Equal(t, test.wantStatusCode, e.Code)

				// nothing more to do, response validation should only be done for '200 OK' responses
				return
			}

			assert.Equal(t, test.wantStatusCode, rec.Result().StatusCode)

			// unpack response
			var balanceResponse rosetta.BalanceResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &balanceResponse))

			// verify that the block data matches the input data
			assert.Equal(t, test.request.BlockID.Index, balanceResponse.BlockID.Index)
			assert.Equal(t, test.request.BlockID.Hash, balanceResponse.BlockID.Hash)

			// verify that we have at least one balance in the response
			if assert.Len(t, balanceResponse.Balances, 1) {

				// verify the balance data - both the value and that the output matches the input spec
				balance := balanceResponse.Balances[0]
				assert.Equal(t, test.request.Currencies[0].Symbol, balance.Currency.Symbol)
				assert.Equal(t, test.request.Currencies[0].Decimals, balance.Currency.Decimals)
				assert.Equal(t, test.wantBalance, balance.Value)
			}
		})
	}
}

// TestGetBalanceBadRequest tests whether an improper JSON (e.g. wrong field types) will cause a '400 Bad Request' error
func TestGetBalanceBadRequest(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	// JSON with an invalid structure (integer instead of string for network name)
	payload := `{ "network_identifier": { "blockchain": "flow", "network": 99} }`

	// create request
	req := httptest.NewRequest(http.MethodPost, "/account/balance", strings.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()

	ctx := echo.New().NewContext(req, rec)

	// execute the request
	err := api.Balance(ctx)
	assert.Error(t, err)

	e, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, e.Code)
}

// getBalanceRequest will generate a BalanceRequest with the specified parameters.
func getBalanceRequest(address string, blockIndex uint64, blockHash string) rosetta.BalanceRequest {

	return rosetta.BalanceRequest{
		NetworkID: getDefaultNetworkID(),
		AccountID: identifier.Account{
			Address: address,
		},
		BlockID: identifier.Block{
			Index: blockIndex,
			Hash:  blockHash,
		},
		Currencies: getDefaultCurrencySpec(),
	}
}

// getDefaultNetworkID returns the Network identifier common for all requests.
func getDefaultNetworkID() identifier.Network {
	return identifier.Network{
		Blockchain: dps.FlowBlockchain,
		Network:    dps.FlowTestnet.String(),
	}
}

// getDefaultCurrencySpec returns the Currency spec common for all requests.
// At the moment only get the FLOW tokens, perhaps in the future it will support multiple.
func getDefaultCurrencySpec() []identifier.Currency {
	return []identifier.Currency{
		{
			Symbol:   dps.FlowSymbol,
			Decimals: dps.FlowDecimals,
		},
	}
}
