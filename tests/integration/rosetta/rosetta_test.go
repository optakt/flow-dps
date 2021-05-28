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
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/models/identifier"
	"github.com/optakt/flow-dps/service/state"

	"github.com/optakt/flow-dps/rosetta/invoker"
	"github.com/optakt/flow-dps/rosetta/retriever"
	"github.com/optakt/flow-dps/rosetta/scripts"
	"github.com/optakt/flow-dps/rosetta/validator"
)

const testChainID = "flow-testnet"

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

	dbSnapshot, err := zstd.NewReader(hex.NewDecoder(strings.NewReader(getCompressedDBSnapshot())))
	if err != nil {
		return nil, nil, err
	}

	defer dbSnapshot.Close()

	err = db.Load(dbSnapshot, 10)
	if err != nil {
		return nil, nil, err
	}

	core, err := state.NewCoreFromDB(db)
	if err != nil {
		return nil, nil, err

	}

	// setup scaffolding for minimal server we need for the tests
	params, ok := dps.FlowParams[flow.ChainID(testChainID)]
	if !ok {
		return nil, nil, fmt.Errorf("invalid chain")
	}

	generator := scripts.NewGenerator(params)
	invoke := invoker.New(zerolog.Nop(), core)
	validate := validator.New(params, core.Height())
	retrieve := retriever.New(generator, invoke)

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
		{
			name:           "valid balance request",
			request:        getBalanceRequest("631e88ae7f1d7c20", 106, "f085d04da7786eb02c16577d8741626c8906cbaece1486b9e3b289d8e2089ae7"),
			wantBalance:    "10000100004",
			wantStatusCode: http.StatusOK,
			wantHandlerErr: assert.NoError,
		},
		{
			name:           "valid balance request 2",
			request:        getBalanceRequest("754aed9de6197641", 106, "f085d04da7786eb02c16577d8741626c8906cbaece1486b9e3b289d8e2089ae7"),
			wantBalance:    "10000099999",
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
			request:        getBalanceRequest("8c5303eaa26202d6", 1, "f085d04da7786eb02c16577d8741626c8906cbaece1486b9e3b289d8e2089ae7"),
			wantStatusCode: http.StatusUnprocessableEntity,
			wantHandlerErr: assert.Error,
		},
		{
			name:           "invalid account address",
			request:        getBalanceRequest("invalid_address", 106, "f085d04da7786eb02c16577d8741626c8906cbaece1486b9e3b289d8e2089ae7"),
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

			// verify that we have precisely one balance in the response
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
		Network:    "flow-testnet",
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
