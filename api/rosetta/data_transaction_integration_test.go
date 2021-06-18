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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/onflow/flow-go/model/flow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/meta"
	"github.com/optakt/flow-dps/rosetta/object"
)

func TestGetTransaction(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	var (
		firstHeader      = knownHeaders(44)
		multipleTxHeader = knownHeaders(165)
		lastHeader       = knownHeaders(181)
	)

	const (
		firstTx = "d5c18baf6c8d11f0693e71dbb951c4856d4f25a456f4d5285a75fd73af39161c"

		// two transactions in a single block
		firstOfTwoTx  = "23c486cfd54bca7138b519203322327bf46e43a780a237d1c5bb0a82f0a06c1d"
		secondOfTwoTx = "3d6922d6c6fd161a76cec23b11067f22cac6409a49b28b905989db64f5cb05a5"

		lastTx = "780bafaf4721ca4270986ea51e659951a8912c2eb99fb1bfedeb753b023cd4d9"
	)

	tests := []struct {
		name string

		request              rosetta.TransactionRequest
		validateTransactions transactionValidationFn
	}{
		{
			// TODO: perhaps do the first ever?
			name:                 "some cherry-picked transaction",
			request:              requestTransaction(firstHeader, firstTx),
			validateTransactions: validateTransfer(t, firstTx, "0x754aed9de6197641", "0x631e88ae7f1d7c20", 1),
		},
		{
			name:                 "first in a block with multiple",
			request:              requestTransaction(multipleTxHeader, firstOfTwoTx),
			validateTransactions: validateTransfer(t, firstOfTwoTx, "0x8c5303eaa26202d6", "0x72157877737ce077", 100_00000000),
		},
		{
			name:                 "second in a block with multiple",
			request:              requestTransaction(multipleTxHeader, secondOfTwoTx),
			validateTransactions: validateTransfer(t, secondOfTwoTx, "0x89c61aa64423504c", "0x82ec283f88a62e65", 1),
		},
		{
			name:                 "last transaction recorded",
			request:              requestTransaction(lastHeader, lastTx),
			validateTransactions: validateTransfer(t, lastTx, "0x668b91e2995c2eba", "0x89c61aa64423504c", 1),
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			// prepare request payload
			enc, err := json.Marshal(test.request)
			require.NoError(t, err)

			// create the request
			req := httptest.NewRequest(http.MethodPost, "/block/transaction", bytes.NewReader(enc))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			rec := httptest.NewRecorder()

			ctx := echo.New().NewContext(req, rec)

			// execute the request
			err = api.Transaction(ctx)
			assert.NoError(t, err)

			assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

			// unpack the response
			var res rosetta.TransactionResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))

			// TODO: change this so it doesn't take an array
			test.validateTransactions([]*object.Transaction{res.Transaction})
		})
	}
}

func TestTransactionErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		testHeight    = 106
		testBlockHash = "1f269f0f45cd2e368e82902d96247113b74da86f6205adf1fd8cf2365418d275"

		trimmedBlockHash = "1f269f0f45cd2e368e82902d96247113b74da86f6205adf1fd8cf2365418d27" // block hash a character short

	)

	var (
		testBlock = identifier.Block{
			Index: testHeight,
			Hash:  testBlockHash,
		}

		// corresponds to the block above
		testTx = identifier.Transaction{Hash: "071e5810f1c8c934aec260f7847400af8f77607ed27ecc02668d7bb2c287c683"}
	)

	tests := []struct {
		name string

		request rosetta.TransactionRequest

		// HTTP/handler errors
		wantStatusCode int

		// rosetta errors - validated separately since it makes reporting mismatches more manageable
		wantRosettaError            meta.ErrorDefinition
		wantRosettaErrorDescription string
		wantRosettaErrorDetails     map[string]interface{}
	}{
		{
			name:    "empty transaction request",
			request: rosetta.TransactionRequest{},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: blockchain field is empty",
		},
		{
			name: "missing blockchain",
			request: rosetta.TransactionRequest{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
				BlockID:       testBlock,
				TransactionID: testTx,
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: blockchain field is empty",
		},
		{
			name: "invalid blockchain",
			request: rosetta.TransactionRequest{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
				BlockID:       testBlock,
				TransactionID: testTx,
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier blockchain (have: %s, want: %s)", invalidBlockchain, dps.FlowBlockchain),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": invalidBlockchain, "network": dps.FlowTestnet.String()},
		},
		{
			name: "missing network",
			request: rosetta.TransactionRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
				BlockID:       testBlock,
				TransactionID: testTx,
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: network field is empty",
		},
		{
			name: "invalid network",
			request: rosetta.TransactionRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    invalidNetwork,
				},
				BlockID:       testBlock,
				TransactionID: testTx,
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier network (have: %s, want: %s)", invalidNetwork, dps.FlowTestnet.String()),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": dps.FlowBlockchain, "network": invalidNetwork},
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

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "block identifier: at least one of hash or index is required",
		},
		{
			name: "wrong length of block id",
			request: rosetta.TransactionRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: 0,
					Hash:  trimmedBlockHash,
				},
				TransactionID: testTx,
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: fmt.Sprintf("block identifier: hash field has wrong length (have: %d, want: %d)", len(trimmedBlockHash), validBlockHashLen),
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			// prepare request payload
			enc, err := json.Marshal(test.request)
			require.NoError(t, err)

			// create the request
			req := httptest.NewRequest(http.MethodPost, "/block/transaction", bytes.NewReader(enc))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			rec := httptest.NewRecorder()

			ctx := echo.New().NewContext(req, rec)

			// execute the request
			err = api.Transaction(ctx)
			assert.Error(t, err)

			echoErr, ok := err.(*echo.HTTPError)
			require.True(t, ok)

			// verify HTTP status code
			assert.Equal(t, test.wantStatusCode, echoErr.Code)

			gotErr, ok := echoErr.Message.(rosetta.Error)
			require.True(t, ok)

			assert.Equal(t, test.wantRosettaError, gotErr.ErrorDefinition)
			assert.Equal(t, test.wantRosettaErrorDescription, gotErr.Description)
			assert.Equal(t, test.wantRosettaErrorDetails, gotErr.Details)
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
