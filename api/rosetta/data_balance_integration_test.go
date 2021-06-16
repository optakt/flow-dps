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

// +build integration

package rosetta_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/invoker"
	"github.com/optakt/flow-dps/rosetta/meta"
	"github.com/optakt/flow-dps/rosetta/retriever"
	"github.com/optakt/flow-dps/rosetta/scripts"
	"github.com/optakt/flow-dps/rosetta/validator"
	"github.com/optakt/flow-dps/service/dictionaries"
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
	dict, _ := hex.DecodeString(dictionaries.Payload)

	decompressor, err := zstd.NewReader(reader,
		zstd.WithDecoderDicts(dict),
	)
	require.NoError(t, err)

	err = db.Load(decompressor, runtime.GOMAXPROCS(0))
	require.NoError(t, err)

	return db
}

func setupAPI(t *testing.T, db *badger.DB) *rosetta.Data {
	t.Helper()

	index := index.NewReader(db)

	params := dps.FlowParams[dps.FlowTestnet]
	config := configuration.New(params.ChainID)
	validate := validator.New(params, index)
	generate := scripts.NewGenerator(params)
	invoke := invoker.New(index)
	retrieve := retriever.New(params, index, validate, generate, invoke)
	controller := rosetta.NewData(config, retrieve)

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
	}{
		{
			name:        "valid balance request - first occurrence of the account",
			request:     balanceRequest("754aed9de6197641", 13, "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"),
			wantBalance: "10000100000",
		},
		{
			name:        "valid balance request - mid chain",
			request:     balanceRequest("754aed9de6197641", 50, "d99888d47dc326fed91087796865316ac71863616f38fa0f735bf1dfab1dc1df"),
			wantBalance: "10000099999",
		},
		{
			name:        "valid balance request - last indexed block",
			request:     balanceRequest("754aed9de6197641", 425, "594d59b2e61bb18b149ffaac2b27b0efe1854f6795cd3bb96a443c3676d78683"),
			wantBalance: "10000100002",
		},
		// TODO: think - what about multiple currencies of the same token? e.g. []token{ "flow", "flow" }?
		// TODO: add a test case that has the currency decimals field unset
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
			assert.NoError(t, err)

			assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

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

func TestBalanceErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	// defined valid balance request fields
	var (
		testAccount = identifier.Account{Address: "754aed9de6197641"}

		testBlock = identifier.Block{
			Index: 13,
			Hash:  "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23",
		}

		validAddressSize = 2 * flow.AddressLength
	)

	// TODO: when blocl tests get merged, this can be global for the tests perhaps
	const (
		invalidBlockchain = "invalid-blockchain"
		invalidNetwork    = "invalid-network"
		invalidToken      = "invalid-token"
		invalidAddress    = "0000000000000000" // valid 16-digit hex value but not a valid account ID

		trimmedBlockHash      = "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb2"  // block hash a character short
		invalidBlockHash      = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz" // invalid hex value
		trimmedAccountAddress = "754aed9de619764"                                                  // account ID a character short
		validLength           = 64
	)

	tests := []struct {
		name string

		request rosetta.BalanceRequest

		wantStatusCode              int
		wantRosettaError            meta.ErrorDefinition
		wantRosettaErrorDescription string
		wantRosettaErrorDetails     map[string]interface{}
	}{
		{
			name:    "empty balance request",
			request: rosetta.BalanceRequest{},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: blockchain field is empty",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "missing network blockchain identifier",
			request: rosetta.BalanceRequest{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: defaultCurrencySpec(),
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: blockchain field is empty",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "wrong network blockchain identifier",
			request: rosetta.BalanceRequest{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: defaultCurrencySpec(),
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier blockchain (have: %s, want: %s)", invalidBlockchain, dps.FlowBlockchain),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": invalidBlockchain, "network": dps.FlowTestnet.String()},
		},
		{
			name: "missing network identifier",
			request: rosetta.BalanceRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: defaultCurrencySpec(),
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: network field is empty",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "wrong network identifier",
			request: rosetta.BalanceRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    invalidNetwork,
				},
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: defaultCurrencySpec(),
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier network (have: %s, want: %s)", invalidNetwork, dps.FlowTestnet.String()),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": dps.FlowBlockchain, "network": invalidNetwork},
		},
		{
			name: "missing block index and height",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 0, Hash: ""},
				Currencies: defaultCurrencySpec(),
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "block identifier: at least one of hash or index is required",
		},
		{
			name: "wrong length of block id",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: trimmedBlockHash},
				Currencies: defaultCurrencySpec(),
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: fmt.Sprintf("block identifier: hash field has wrong length (have: %d, want: %d)", len(trimmedBlockHash), validLength),
		},
		{
			name: "missing account address",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  identifier.Account{Address: ""},
				BlockID:    testBlock,
				Currencies: defaultCurrencySpec(),
			},
			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "account identifier: address field is empty",
		},
		{
			name: "wrong length of account address",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  identifier.Account{Address: trimmedAccountAddress},
				BlockID:    testBlock,
				Currencies: defaultCurrencySpec(),
			},
			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: fmt.Sprintf("account identifier: address field has wrong length (have: %d, want: %d)", len(trimmedAccountAddress), validAddressSize),
		},
		{
			name: "missing currency data",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: []identifier.Currency{},
			},
			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "currency identifiers: currency list is empty",
		},
		{
			name: "missing currency symbol",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: []identifier.Currency{{Symbol: "", Decimals: 8}},
			},
			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "currency identifier: symbol field is missing",
		},
		{
			name: "some currency symbols missing",
			request: rosetta.BalanceRequest{
				NetworkID: defaultNetworkID(),
				AccountID: testAccount,
				BlockID:   testBlock,
				Currencies: []identifier.Currency{
					{Symbol: dps.FlowSymbol, Decimals: 8},
					{Symbol: "", Decimals: 8},
					{Symbol: dps.FlowSymbol, Decimals: 8},
				},
			},
			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "currency identifier: symbol field is missing",
		},
		{
			name: "missing block height",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Hash: "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"},
				Currencies: defaultCurrencySpec(),
			},
			wantStatusCode:              http.StatusInternalServerError,
			wantRosettaError:            configuration.ErrorInternal,
			wantRosettaErrorDescription: "could not validate block: block access with hash currently not supported",
		},
		{
			name: "invalid block hash",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: invalidBlockHash},
				Currencies: defaultCurrencySpec(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidBlock,
			wantRosettaErrorDescription: "block hash is not a valid hex-encoded string",
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(13), "hash": invalidBlockHash},
		},
		{
			name: "unkown block requested",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 426},
				Currencies: defaultCurrencySpec(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorUnknownBlock,
			wantRosettaErrorDescription: "block index is above last indexed block (last: 425)",
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(426), "hash": ""},
		},
		{
			name: "mismatched block id and height",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: "9035c558379b208eba11130c928537fe50ad93cdee314980fccb695aa31df7fc"},
				Currencies: defaultCurrencySpec(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidBlock,
			wantRosettaErrorDescription: "block hash does not match known hash for height (known: af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23)",
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(13), "hash": "9035c558379b208eba11130c928537fe50ad93cdee314980fccb695aa31df7fc"},
		},
		{
			name: "invalid account ID hex",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  identifier.Account{Address: "zzzzzzzzzzzzzzzz"},
				BlockID:    testBlock,
				Currencies: defaultCurrencySpec(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidAccount,
			wantRosettaErrorDescription: "account address is not a valid hex-encoded string",
			wantRosettaErrorDetails:     map[string]interface{}{"address": "zzzzzzzzzzzzzzzz", "chain": dps.FlowTestnet.String()},
		},
		{
			name: "invalid account ID",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  identifier.Account{Address: invalidAddress},
				BlockID:    testBlock,
				Currencies: defaultCurrencySpec(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidAccount,
			wantRosettaErrorDescription: "account address is not valid for configured chain",
			wantRosettaErrorDetails:     map[string]interface{}{"address": invalidAddress, "chain": dps.FlowTestnet.String()},
		},
		{
			name: "unknown currency requested",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: []identifier.Currency{{Symbol: invalidToken, Decimals: 8}},
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorUnknownCurrency,
			wantRosettaErrorDescription: "currency symbol has not been configured",
			wantRosettaErrorDetails:     map[string]interface{}{"symbol": invalidToken, "decimals": uint(8)},
		},
		{
			name: "invalid currency decimal count",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetworkID(),
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: []identifier.Currency{{Symbol: dps.FlowSymbol, Decimals: 7}},
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidCurrency,
			wantRosettaErrorDescription: fmt.Sprintf("currency decimals do not match configured default (default: %d)", dps.FlowDecimals),
			wantRosettaErrorDetails:     map[string]interface{}{"symbol": dps.FlowSymbol, "decimals": uint(7)},
		},
		{
			name: "invalid currency decimal count in a list of currencies",
			request: rosetta.BalanceRequest{
				NetworkID: defaultNetworkID(),
				AccountID: testAccount,
				BlockID:   testBlock,
				Currencies: []identifier.Currency{
					{Symbol: dps.FlowSymbol, Decimals: dps.FlowDecimals},
					{Symbol: dps.FlowSymbol, Decimals: 7},
					{Symbol: dps.FlowSymbol, Decimals: dps.FlowDecimals},
				},
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidCurrency,
			wantRosettaErrorDescription: fmt.Sprintf("currency decimals do not match configured default (default: %d)", dps.FlowDecimals),
			wantRosettaErrorDetails:     map[string]interface{}{"symbol": dps.FlowSymbol, "decimals": uint(7)},
		},
		// TODO: request account balance at height where it does not exist
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			// TODO: move this code to a helper function
			enc, err := json.Marshal(test.request)
			require.NoError(t, err)

			// create request
			req := httptest.NewRequest(http.MethodPost, "/account/balance", bytes.NewReader(enc))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			rec := httptest.NewRecorder()

			ctx := echo.New().NewContext(req, rec)

			// execute the request
			err = api.Balance(ctx)
			assert.Error(t, err)

			echoErr, ok := err.(*echo.HTTPError)
			require.True(t, ok)

			assert.Equal(t, test.wantStatusCode, echoErr.Code)
			gotErr, ok := echoErr.Message.(rosetta.Error)
			require.True(t, ok)

			assert.Equal(t, test.wantRosettaError, gotErr.ErrorDefinition)
			assert.Equal(t, test.wantRosettaErrorDescription, gotErr.Description)
			assert.Equal(t, test.wantRosettaErrorDetails, gotErr.Details)
		})
	}
}

// TestMalformedBalanceRequest tests whether an improper JSON (e.g. wrong field types) will cause a '400 Bad Request' error
func TestMalformedBalanceRequest(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		wrongFieldType = `{
			"network_identifier": {
				"blockchain": "flow",
				"network": 99
			}
		}`

		unclosedBracket = `{
			"network_identifier": {
				"blockchain" : "flow",
				"network" : "flow-testnet"
			},
			"block_identifier" : {
				"index" : 13,
				"hash" : "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"
			},
			"account_identifier" : {
				"address" : "754aed9de6197641"
			},
			"currencies" : [
				{ "symbol" : "FLOW" , "decimals" : 8 }
			]`

		validJSON = `{
			"network_identifier": {
				"blockchain" : "flow",
				"network" : "flow-testnet"
			},
			"block_identifier" : {
				"index" : 13,
				"hash" : "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"
			},
			"account_identifier" : {
				"address" : "754aed9de6197641"
			},
			"currencies" : [
				{ "symbol" : "FLOW" , "decimals" : 8 }
			]
		}`
	)

	tests := []struct {
		name     string
		payload  []byte
		mimeType string
	}{
		{
			name:     "wrong field type",
			payload:  []byte(wrongFieldType),
			mimeType: echo.MIMEApplicationJSON,
		},
		{
			name:     "unclosed bracket",
			payload:  []byte(unclosedBracket),
			mimeType: echo.MIMEApplicationJSON,
		},
		{
			name:     "valid payload with no mime type set",
			payload:  []byte(validJSON),
			mimeType: "",
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			// create request
			req := httptest.NewRequest(http.MethodPost, "/account/balance", bytes.NewReader(test.payload))
			req.Header.Set(echo.HeaderContentType, test.mimeType)

			rec := httptest.NewRecorder()

			ctx := echo.New().NewContext(req, rec)

			// execute the request
			err := api.Balance(ctx)

			// verify the errors

			assert.Error(t, err)

			echoErr, ok := err.(*echo.HTTPError)
			require.True(t, ok)

			assert.Equal(t, http.StatusBadRequest, echoErr.Code)
			gotErr, ok := echoErr.Message.(rosetta.Error)
			require.True(t, ok)

			assert.Equal(t, configuration.ErrorInvalidFormat, gotErr.ErrorDefinition)
		})
	}
}

// balanceRequest generates a BalanceRequest with the specified parameters.
func balanceRequest(address string, blockIndex uint64, blockHash string) rosetta.BalanceRequest {

	return rosetta.BalanceRequest{
		NetworkID: defaultNetworkID(),
		AccountID: identifier.Account{
			Address: address,
		},
		BlockID: identifier.Block{
			Index: blockIndex,
			Hash:  blockHash,
		},
		Currencies: defaultCurrencySpec(),
	}
}

// defaultNetworkID returns the Network identifier common for all requests.
func defaultNetworkID() identifier.Network {
	return identifier.Network{
		Blockchain: dps.FlowBlockchain,
		Network:    dps.FlowTestnet.String(),
	}
}

// defaultCurrencySpec returns the Currency spec common for all requests.
// At the moment only get the FLOW tokens, perhaps in the future it will support multiple.
func defaultCurrencySpec() []identifier.Currency {
	return []identifier.Currency{
		{
			Symbol:   dps.FlowSymbol,
			Decimals: dps.FlowDecimals,
		},
	}
}
