// Copyright 2021 Optakt Labs OÜ
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

const (
	invalidBlockchain = "invalid-blockchain"
	invalidNetwork    = "invalid-network"
	invalidToken      = "invalid-token"

	invalidBlockHash = "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb2z" // invalid hex value

	validBlockHashLen = 64
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
	invoke, err := invoker.New(index)
	require.NoError(t, err)
	retrieve := retriever.New(params, index, validate, generate, invoke)
	controller := rosetta.NewData(config, retrieve)

	return controller
}
func TestGetBalance(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		testAccount = "754aed9de6197641"
	)

	var (
		// block where the account first appears
		firstBlock = identifier.Block{
			Index: 13,
			Hash:  "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23",
		}

		// a block mid-chain
		secondBlock = identifier.Block{
			Index: 50,
			Hash:  "d99888d47dc326fed91087796865316ac71863616f38fa0f735bf1dfab1dc1df",
		}

		// last indexed block
		lastBlock = identifier.Block{
			Index: 425,
			Hash:  "594d59b2e61bb18b149ffaac2b27b0efe1854f6795cd3bb96a443c3676d78683",
		}
	)

	tests := []struct {
		name string

		request rosetta.BalanceRequest

		wantStatusCode int
		wantBalance    string
		validateBlock  blockIDValidationFn
	}{
		{
			name:          "first occurrence of the account",
			request:       requestBalance(testAccount, firstBlock),
			wantBalance:   "10000100000",
			validateBlock: validateBlock(t, firstBlock.Index, firstBlock.Hash),
		},
		{
			name:          "mid chain",
			request:       requestBalance(testAccount, secondBlock),
			wantBalance:   "10000099999",
			validateBlock: validateBlock(t, secondBlock.Index, secondBlock.Hash),
		},
		{
			name:          "last indexed block",
			request:       requestBalance(testAccount, lastBlock),
			wantBalance:   "10000100002",
			validateBlock: validateBlock(t, lastBlock.Index, lastBlock.Hash),
		},
		{
			// use block height only to retrieve data, but verify hash is set in the response
			name:          "get block via height only",
			request:       requestBalance(testAccount, identifier.Block{Index: secondBlock.Index}),
			wantBalance:   "10000099999",
			validateBlock: validateBlock(t, secondBlock.Index, secondBlock.Hash),
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
			assert.NoError(t, err)

			assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

			// unpack response
			var balanceResponse rosetta.BalanceResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &balanceResponse))

			test.validateBlock(balanceResponse.BlockID)

			// verify that we have one balance in the response
			if assert.Len(t, balanceResponse.Balances, 1) {

				// verify returned balance - both the value and that the output matches the input spec
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

	const (
		invalidAddress = "0000000000000000" // valid 16-digit hex value but not a valid account ID

		trimmedBlockHash = "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb2" // block hash a character short

		trimmedAddress    = "754aed9de619764"  // account ID a character short
		invalidAddressHex = "754aed9de619764z" // invalid hex string

		accFirstOccurrence = 13

		cadenceVaultNotFoundErr = "could not invoke script: script execution encountered error: [Error Code: 1101] cadence runtime error Execution failed:\nerror: panic: Could not borrow Balance reference to the Vault\n  --> d6b84b6f36db7d880d4ecc2a6a952094301873657e204e86d4cc9282c0df4b3d:11:11\n   |\n11 |         ?? panic(\"Could not borrow Balance reference to the Vault\")\n   |            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"
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
				Currencies: defaultCurrency(),
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
				Currencies: defaultCurrency(),
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
				Currencies: defaultCurrency(),
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
				Currencies: defaultCurrency(),
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier network (have: %s, want: %s)", invalidNetwork, dps.FlowTestnet.String()),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": dps.FlowBlockchain, "network": invalidNetwork},
		},
		{
			name: "missing block index and height",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 0, Hash: ""},
				Currencies: defaultCurrency(),
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "block identifier: at least one of hash or index is required",
		},
		{
			name: "wrong length of block id",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: trimmedBlockHash},
				Currencies: defaultCurrency(),
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: fmt.Sprintf("block identifier: hash field has wrong length (have: %d, want: %d)", len(trimmedBlockHash), validBlockHashLen),
		},
		{
			name: "missing account address",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  identifier.Account{Address: ""},
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "account identifier: address field is empty",
		},
		{
			name: "wrong length of account address",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  identifier.Account{Address: trimmedAddress},
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: fmt.Sprintf("account identifier: address field has wrong length (have: %d, want: %d)", len(trimmedAddress), validAddressSize),
		},
		{
			name: "missing currency data",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
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
				NetworkID:  defaultNetwork(),
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
				NetworkID: defaultNetwork(),
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
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Hash: "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"},
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusInternalServerError,
			wantRosettaError:            configuration.ErrorInternal,
			wantRosettaErrorDescription: "could not validate block: block access with hash currently not supported",
		},
		{
			name: "invalid block hash",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: invalidBlockHash},
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidBlock,
			wantRosettaErrorDescription: "block hash is not a valid hex-encoded string",
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(13), "hash": invalidBlockHash},
		},
		{
			name: "unkown block requested",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 426},
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorUnknownBlock,
			wantRosettaErrorDescription: "block index is above last indexed block (last: 425)",
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(426), "hash": ""},
		},
		{
			name: "mismatched block id and height",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: "9035c558379b208eba11130c928537fe50ad93cdee314980fccb695aa31df7fc"},
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidBlock,
			wantRosettaErrorDescription: "block hash does not match known hash for height (known: af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23)",
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(13), "hash": "9035c558379b208eba11130c928537fe50ad93cdee314980fccb695aa31df7fc"},
		},
		{
			name: "invalid account ID hex",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  identifier.Account{Address: invalidAddressHex},
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidAccount,
			wantRosettaErrorDescription: "account address is not a valid hex-encoded string",
			wantRosettaErrorDetails:     map[string]interface{}{"address": invalidAddressHex, "chain": dps.FlowTestnet.String()},
		},
		{
			name: "invalid account ID",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  identifier.Account{Address: invalidAddress},
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidAccount,
			wantRosettaErrorDescription: "account address is not valid for configured chain",
			wantRosettaErrorDetails:     map[string]interface{}{"address": invalidAddress, "chain": dps.FlowTestnet.String()},
		},
		{
			name: "unknown currency requested",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
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
				NetworkID:  defaultNetwork(),
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
				NetworkID: defaultNetwork(),
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
		{
			// request the account balance before the account is created, so the account/vault does not exist.
			// TODO: differentiate between vault and account not existing
			// => https://github.com/optakt/flow-dps/issues/204
			name: "account vault does not exist",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: accFirstOccurrence - 1}, // one block before the account is created
				Currencies: defaultCurrency(),
			},
			wantStatusCode:              http.StatusInternalServerError,
			wantRosettaError:            configuration.ErrorInternal,
			wantRosettaErrorDescription: cadenceVaultNotFoundErr,
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

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

// requestBalance generates a BalanceRequest with the specified parameters.
func requestBalance(address string, id identifier.Block) rosetta.BalanceRequest {

	return rosetta.BalanceRequest{
		NetworkID: defaultNetwork(),
		AccountID: identifier.Account{
			Address: address,
		},
		BlockID:    id,
		Currencies: defaultCurrency(),
	}
}

// defaultNetwork returns the Network identifier common for all requests.
func defaultNetwork() identifier.Network {
	return identifier.Network{
		Blockchain: dps.FlowBlockchain,
		Network:    dps.FlowTestnet.String(),
	}
}

// defaultCurrency returns the Currency spec common for all requests.
// At the moment only get the FLOW tokens, perhaps in the future it will support multiple.
func defaultCurrency() []identifier.Currency {
	return []identifier.Currency{
		{
			Symbol:   dps.FlowSymbol,
			Decimals: dps.FlowDecimals,
		},
	}
}
