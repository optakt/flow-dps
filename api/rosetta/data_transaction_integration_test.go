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

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

func TestAPI_Transaction(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	var (
		firstHeader      = knownHeader(44)
		multipleTxHeader = knownHeader(165)
		lastHeader       = knownHeader(181)

		// two transactions in a single block
		midBlockTxs = []string{
			"23c486cfd54bca7138b519203322327bf46e43a780a237d1c5bb0a82f0a06c1d",
			"3d6922d6c6fd161a76cec23b11067f22cac6409a49b28b905989db64f5cb05a5",
		}
	)

	const (
		firstTx = "d5c18baf6c8d11f0693e71dbb951c4856d4f25a456f4d5285a75fd73af39161c"
		lastTx  = "780bafaf4721ca4270986ea51e659951a8912c2eb99fb1bfedeb753b023cd4d9"
	)

	tests := []struct {
		name string

		request    rosetta.TransactionRequest
		validateTx validateTxFunc
	}{
		{
			name:       "some cherry picked transaction",
			request:    requestTransaction(firstHeader, firstTx),
			validateTx: validateTransfer(t, firstTx, "754aed9de6197641", "631e88ae7f1d7c20", 1),
		},
		{
			name:       "first in a block with multiple",
			request:    requestTransaction(multipleTxHeader, midBlockTxs[0]),
			validateTx: validateTransfer(t, midBlockTxs[0], "8c5303eaa26202d6", "72157877737ce077", 100_00000000),
		},
		{
			// The test does not have blocks with more than two transactions, so this is the same as 'get the last transaction from a block'.
			name:       "second in a block with multiple",
			request:    requestTransaction(multipleTxHeader, midBlockTxs[1]),
			validateTx: validateTransfer(t, midBlockTxs[1], "89c61aa64423504c", "82ec283f88a62e65", 1),
		},
		{
			name:       "last transaction recorded",
			request:    requestTransaction(lastHeader, lastTx),
			validateTx: validateTransfer(t, lastTx, "668b91e2995c2eba", "89c61aa64423504c", 1),
		},
		{
			name: "lookup using height and transaction hash",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: 165,
				},
				TransactionID: identifier.Transaction{
					Hash: midBlockTxs[1],
				},
			},
			validateTx: validateTransfer(t, midBlockTxs[1], "89c61aa64423504c", "82ec283f88a62e65", 1),
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			rec, ctx, err := setupRecorder(transactionEndpoint, test.request)
			require.NoError(t, err)

			err = api.Transaction(ctx)
			assert.NoError(t, err)

			assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

			var res rosetta.TransactionResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))

			test.validateTx([]*object.Transaction{res.Transaction})
		})
	}
}

func TestAPI_TransactionHandlesErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		lastHeight = 425

		testHeight    = 106
		testBlockHash = "1f269f0f45cd2e368e82902d96247113b74da86f6205adf1fd8cf2365418d275"
		testTxHash    = "071e5810f1c8c934aec260f7847400af8f77607ed27ecc02668d7bb2c287c683"

		trimmedBlockHash = "1f269f0f45cd2e368e82902d96247113b74da86f6205adf1fd8cf2365418d27"  // block hash a character short
		trimmedTxHash    = "071e5810f1c8c934aec260f7847400af8f77607ed27ecc02668d7bb2c287c68"  // tx hash a character short
		invalidTxHash    = "071e5810f1c8c934aec260f7847400af8f77607ed27ecc02668d7bb2c287c68z" // testTxHash with a hex-invalid last character
		unknownTxHash    = "602dd6b7fad80b0e6869eaafd55625faa16341f09027dc925a8e8cef267e5683" // tx from another block
	)

	var (
		testBlock = identifier.Block{
			Index: testHeight,
			Hash:  testBlockHash,
		}

		// corresponds to the block above
		testTx = identifier.Transaction{Hash: testTxHash}
	)

	tests := []struct {
		name string

		request rosetta.TransactionRequest

		checkErr assert.ErrorAssertionFunc
	}{
		{
			name:    "empty transaction request",
			request: rosetta.TransactionRequest{},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing blockchain name",
			request: rosetta.TransactionRequest{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
				BlockID:       testBlock,
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid blockchain name",
			request: rosetta.TransactionRequest{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
				BlockID:       testBlock,
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
		{
			name: "missing network name",
			request: rosetta.TransactionRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
				BlockID:       testBlock,
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid network name",
			request: rosetta.TransactionRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    invalidNetwork,
				},
				BlockID:       testBlock,
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
		{
			name: "missing block height and hash",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: 0,
					Hash:  "",
				},
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid length of block id",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: testHeight,
					Hash:  trimmedBlockHash,
				},
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing block height",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Hash: testBlockHash,
				},
				TransactionID: testTx,
			},
			checkErr: checkRosettaError(http.StatusInternalServerError, configuration.ErrorInternal),
		},
		{
			name: "invalid block hash",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: testHeight,
					Hash:  invalidBlockHash,
				},
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidBlock),
		},
		{
			name: "unknown block",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: lastHeight + 1,
				},
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorUnknownBlock),
		},
		{
			name: "mismatched block height and hash",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: 44,
					Hash:  testBlockHash,
				},
				TransactionID: testTx,
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidBlock),
		},
		{
			name: "missing transaction id",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID:   testBlock,
				TransactionID: identifier.Transaction{
					Hash: "",
				},
			},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing transaction id",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID:   testBlock,
				TransactionID: identifier.Transaction{
					Hash: trimmedTxHash,
				},
			},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid transaction id",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID:   testBlock,
				TransactionID: identifier.Transaction{
					Hash: invalidTxHash,
				},
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidTransaction),
		},
		// TODO: add - transaction that has no events/transfers
		{
			name: "transaction missing from block",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				TransactionID: identifier.Transaction{
					Hash: unknownTxHash,
				},
				BlockID: testBlock,
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorUnknownTransaction),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, ctx, err := setupRecorder(transactionEndpoint, test.request)
			require.NoError(t, err)

			err = api.Transaction(ctx)
			test.checkErr(t, err)
		})
	}
}

func TestAPI_TransactionHandlesMalformedRequest(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		// network field is an integer instead of a string
		wrongFieldType = `
		{ 
			"network_identifier": { 
				"blockchain": "flow", 
				"network": 99
			}
		}`

		unclosedBracket = `
		{
			"network_identifier" : {
				"blockchain": "flow",
				"network": "flow-testnet"
			},
			"block_identifier": {
				"index": 106,
				"hash": "1f269f0f45cd2e368e82902d96247113b74da86f6205adf1fd8cf2365418d275"
			},
			"transaction_identifier": {
				"hash": "071e5810f1c8c934aec260f7847400af8f77607ed27ecc02668d7bb2c287c683"
			}`

		validJSON = `
		{
			"network_identifier" : {
				"blockchain": "flow",
				"network": "flow-testnet"
			},
			"block_identifier": {
				"index": 106,
				"hash": "1f269f0f45cd2e368e82902d96247113b74da86f6205adf1fd8cf2365418d275"
			},
			"transaction_identifier": {
				"hash": "071e5810f1c8c934aec260f7847400af8f77607ed27ecc02668d7bb2c287c683"
			}
		}`
	)

	tests := []struct {
		name    string
		payload []byte
		prepare func(*http.Request)
	}{
		{
			name:    "empty request",
			payload: []byte(``),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			},
		},
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

			_, ctx, err := setupRecorder(transactionEndpoint, test.payload, test.prepare)
			require.NoError(t, err)

			err = api.Block(ctx)
			assert.Error(t, err)

			echoErr, ok := err.(*echo.HTTPError)
			require.True(t, ok)

			assert.Equal(t, http.StatusBadRequest, echoErr.Code)

			gotErr, ok := echoErr.Message.(rosetta.Error)
			require.True(t, ok)

			assert.Equal(t, configuration.ErrorInvalidFormat, gotErr.ErrorDefinition)
			assert.NotEmpty(t, gotErr.Description)
		})
	}

}

func requestTransaction(header flow.Header, txID string) rosetta.TransactionRequest {

	return rosetta.TransactionRequest{
		NetworkID: defaultNetwork(),
		BlockID: identifier.Block{
			Index: header.Height,
			Hash:  header.ID().String(),
		},
		TransactionID: identifier.Transaction{
			Hash: txID,
		},
	}
}
