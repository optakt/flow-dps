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
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/identifier"
	"github.com/optakt/flow-dps/service/state"

	"github.com/optakt/flow-dps/rosetta/invoker"
	"github.com/optakt/flow-dps/rosetta/retriever"
	"github.com/optakt/flow-dps/rosetta/validator"
)

// initialize the rosetta Data HTTP handler once and reuse for all tests.
var rosettaSvc *rosetta.Data

func setupDB() (*rosetta.Data, *badger.DB, error) {

	const testDbBackupFile = "rosetta-get-balance-test.db"
	opts := badger.DefaultOptions("").
		WithInMemory(true).
		WithReadOnly(true).
		WithLogger(nil)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, nil, err
	}

	dbSnapshot, err := os.Open(testDbBackupFile)
	if err != nil {
		return nil, nil, err

	}

	err = db.Load(dbSnapshot, 10)
	if err != nil {
		return nil, nil, err

	}

	core, err := state.NewCoreFromDB(db)
	if err != nil {
		return nil, nil, err

	}

	// setup scaffolding for minimal server we need for the tests
	chain := flow.ChainID("flow-testnet").Chain()
	validate := validator.New(chain, core.Height())
	headers := invoker.NewHeaders(core.Chain())
	invoke := invoker.New(zerolog.Nop(), core, chain, headers)
	retrieve := retriever.New(invoke)

	svc := rosetta.NewData(validate, retrieve)

	return svc, db, err
}

func TestMain(m *testing.M) {

	svc, db, err := setupDB()
	if err != nil {
		log.Fatalf("could not perform setup for test: %v", err)
	}

	rosettaSvc = svc

	defer db.Close()

	m.Run()
}

func TestGetBalance(t *testing.T) {

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
			err = rosettaSvc.Balance(ctx)
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
			assert.Len(t, balanceResponse.Balances, 1)

			if len(balanceResponse.Balances) > 0 {

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

	// JSON with an invalid structure (integer instead of string for network name)
	payload := `{ "network_identifier": { "blockchain": "flow", "network": 99} }`

	// create request
	req := httptest.NewRequest(http.MethodPost, "/account/balance", strings.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()

	ctx := echo.New().NewContext(req, rec)

	// execute the request
	err := rosettaSvc.Balance(ctx)
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
		Blockchain: "flow",
		Network:    "testnet",
	}
}

// getDefaultCurrencySpec returns the Currency spec common for all requests.
// At the moment only get the FLOW tokens, perhaps in the future it will support multiple.
func getDefaultCurrencySpec() []identifier.Currency {
	return []identifier.Currency{
		{
			Symbol:   "FLOW",
			Decimals: 8,
		},
	}
}
