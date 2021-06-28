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
	"encoding/json"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

func TestAPI_Balance(t *testing.T) {
	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		testAccount = "754aed9de6197641"
	)

	var (
		// Block where the account first appears.
		firstBlock = identifier.Block{
			Index: 13,
			Hash:  "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23",
		}

		// A block mid-chain.
		secondBlock = identifier.Block{
			Index: 50,
			Hash:  "d99888d47dc326fed91087796865316ac71863616f38fa0f735bf1dfab1dc1df",
		}

		// Last indexed block.
		lastBlock = identifier.Block{
			Index: 425,
			Hash:  "594d59b2e61bb18b149ffaac2b27b0efe1854f6795cd3bb96a443c3676d78683",
		}
	)

	tests := []struct {
		name string

		request rosetta.BalanceRequest

		wantBalance   string
		validateBlock validateBlockFunc
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
			// Use block height only to retrieve data, but verify hash is set in the response.
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

			rec, ctx, err := setupRecorder(balanceEndpoint, test.request)
			require.NoError(t, err)

			err = api.Balance(ctx)
			assert.NoError(t, err)

			assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

			var balanceResponse rosetta.BalanceResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &balanceResponse))

			test.validateBlock(balanceResponse.BlockID)

			if assert.Len(t, balanceResponse.Balances, 1) {
				balance := balanceResponse.Balances[0]

				assert.Equal(t, test.request.Currencies[0].Symbol, balance.Currency.Symbol)
				assert.Equal(t, test.request.Currencies[0].Decimals, balance.Currency.Decimals)
				assert.Equal(t, test.wantBalance, balance.Value)
			}
		})
	}
}

func TestAPI_BalanceHandlesErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	// Defined valid balance request fields.
	var (
		testAccount = identifier.Account{Address: "754aed9de6197641"}

		testBlock = identifier.Block{
			Index: 13,
			Hash:  "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23",
		}
	)

	const (
		invalidAddress = "0000000000000000" // valid 16-digit hex value but not a valid account ID

		trimmedBlockHash = "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb2" // block hash a character short

		trimmedAddress    = "754aed9de619764"  // account ID a character short
		invalidAddressHex = "754aed9de619764z" // invalid hex string

		accFirstOccurrence = 13
	)

	tests := []struct {
		name string

		request rosetta.BalanceRequest

		checkError assert.ErrorAssertionFunc
	}{
		{
			name:    "empty balance request",
			request: rosetta.BalanceRequest{},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing blockchain name",
			request: rosetta.BalanceRequest{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid blockchain name",
			request: rosetta.BalanceRequest{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
		{
			name: "missing network name",
			request: rosetta.BalanceRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},
			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid network name",
			request: rosetta.BalanceRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    invalidNetwork,
				},
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
		{
			name: "missing block index and height",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 0, Hash: ""},
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid length of block id",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: trimmedBlockHash},
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing account address",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  identifier.Account{Address: ""},
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid length of account address",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  identifier.Account{Address: trimmedAddress},
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing currency data",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: []identifier.Currency{},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing currency symbol",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: []identifier.Currency{{Symbol: "", Decimals: 8}},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
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

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing block height",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Hash: "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"},
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusInternalServerError, configuration.ErrorInternal),
		},
		{
			name: "invalid block hash",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: invalidBlockHash},
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidBlock),
		},
		{
			name: "unkown block requested",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 426},
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorUnknownBlock),
		},
		{
			name: "mismatched block id and height",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: 13, Hash: "9035c558379b208eba11130c928537fe50ad93cdee314980fccb695aa31df7fc"},
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidBlock),
		},
		{
			name: "invalid account ID hex",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  identifier.Account{Address: invalidAddressHex},
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidAccount),
		},
		{
			name: "invalid account ID",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  identifier.Account{Address: invalidAddress},
				BlockID:    testBlock,
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidAccount),
		},
		{
			name: "unknown currency requested",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: []identifier.Currency{{Symbol: invalidToken, Decimals: 8}},
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorUnknownCurrency),
		},
		{
			name: "invalid currency decimal count",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    testBlock,
				Currencies: []identifier.Currency{{Symbol: dps.FlowSymbol, Decimals: 7}},
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidCurrency),
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

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidCurrency),
		},
		{
			// Request the account balance before the account is created, so the account/vault does not exist.
			// TODO: differentiate between vault and account not existing
			// => https://github.com/optakt/flow-dps/issues/204
			name: "account vault does not exist",
			request: rosetta.BalanceRequest{
				NetworkID:  defaultNetwork(),
				AccountID:  testAccount,
				BlockID:    identifier.Block{Index: accFirstOccurrence - 1}, // one block before the account is created
				Currencies: defaultCurrency(),
			},

			checkError: checkRosettaError(http.StatusInternalServerError, configuration.ErrorInternal),
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			_, ctx, err := setupRecorder(balanceEndpoint, test.request)
			require.NoError(t, err)

			// Execute the request.
			err = api.Balance(ctx)
			test.checkError(t, err)
		})
	}
}

// TestAPI_BalanceHandlesMalformedRequest tests whether an improper JSON (e.g. wrong field types) results in a '400 Bad Request' error.
func TestAPI_BalanceHandlesMalformedRequest(t *testing.T) {

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
		name    string
		payload []byte
		prepare func(req *http.Request)
	}{
		{
			name:    "wrong field type",
			payload: []byte(wrongFieldType),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			},
		},
		{
			name:    "unclosed bracket",
			payload: []byte(unclosedBracket),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			},
		},
		{
			name:    "valid payload with no MIME type set",
			payload: []byte(validJSON),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, "")
			},
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			_, ctx, err := setupRecorder(balanceEndpoint, test.payload, test.prepare)
			require.NoError(t, err)

			err = api.Balance(ctx)

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
