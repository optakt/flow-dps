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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/onflow/flow-go/model/flow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/meta"
	"github.com/optakt/flow-dps/rosetta/object"
)

type validateBlockFunc func(identifier.Block)
type validateTxFunc func([]*object.Transaction)

func TestAPI_Block(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	// headers of known blocks we want to verify
	var (
		firstHeader = knownHeaders(1)
		midHeader1  = knownHeaders(13)
		midHeader2  = knownHeaders(43)
		midHeader3  = knownHeaders(44)
		lastHeader  = knownHeaders(425) // header of last indexed block
	)

	const (
		rootAccount     = "8c5303eaa26202d6"
		senderAccount   = "754aed9de6197641"
		receiverAccount = "631e88ae7f1d7c20"

		initialLoadTx = "a9c9ab28ea76b7dbfd1f2666f74348e4188d67cf68248df6634cee3f06adf7b1"
		transferTx    = "d5c18baf6c8d11f0693e71dbb951c4856d4f25a456f4d5285a75fd73af39161c"
	)

	tests := []struct {
		name string

		request rosetta.BlockRequest

		wantTimestamp        int64
		wantParentHash       string
		validateTransactions validateTxFunc
		validateBlock        validateBlockFunc
	}{
		{
			name:    "child of first block",
			request: blockRequest(firstHeader),

			wantTimestamp:  convert.RosettaTime(firstHeader.Timestamp),
			wantParentHash: firstHeader.ParentID.String(),
			validateBlock:  validateByHeader(t, firstHeader),
		},
		{
			// initial transfer of currency from the root account to the user - 100 tokens
			name:    "block mid-chain with transactions",
			request: blockRequest(midHeader1),

			wantTimestamp:        convert.RosettaTime(midHeader1.Timestamp),
			wantParentHash:       midHeader1.ParentID.String(),
			validateBlock:        validateByHeader(t, midHeader1),
			validateTransactions: validateTransfer(t, initialLoadTx, rootAccount, senderAccount, 100_00000000),
		},
		{
			name:    "block mid-chain without transactions",
			request: blockRequest(midHeader2),

			wantTimestamp:  convert.RosettaTime(midHeader2.Timestamp),
			validateBlock:  validateByHeader(t, midHeader2),
			wantParentHash: midHeader2.ParentID.String(),
		},
		{
			// transaction between two users
			name:    "second block mid-chain with transactions",
			request: blockRequest(midHeader3),

			wantTimestamp:        convert.RosettaTime(midHeader3.Timestamp),
			wantParentHash:       midHeader3.ParentID.String(),
			validateBlock:        validateByHeader(t, midHeader3),
			validateTransactions: validateTransfer(t, transferTx, senderAccount, receiverAccount, 1),
		},
		{
			name: "lookup of a block mid-chain by index only",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID:   identifier.Block{Index: midHeader3.Height},
			},

			wantTimestamp:        convert.RosettaTime(midHeader3.Timestamp),
			wantParentHash:       midHeader3.ParentID.String(),
			validateTransactions: validateTransfer(t, transferTx, senderAccount, receiverAccount, 1),
			validateBlock:        validateBlock(t, midHeader3.Height, midHeader3.ID().String()), // verify that the returned block ID has both height and hash
		},
		{
			name:    "last indexed block",
			request: blockRequest(lastHeader),

			wantTimestamp:  convert.RosettaTime(lastHeader.Timestamp),
			validateBlock:  validateByHeader(t, lastHeader),
			wantParentHash: lastHeader.ParentID.String(),
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			enc, err := json.Marshal(test.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/block", bytes.NewReader(enc))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			rec := httptest.NewRecorder()

			ctx := echo.New().NewContext(req, rec)

			err = api.Block(ctx)
			assert.NoError(t, err)

			// unpack response
			var blockResponse rosetta.BlockResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &blockResponse))
			require.NotNil(t, blockResponse.Block)

			// validate index/hash of returned block
			test.validateBlock(blockResponse.Block.ID)

			// verify the index/hash of the parent block
			assert.Equal(t, test.request.BlockID.Index-1, blockResponse.Block.ParentID.Index)
			assert.Equal(t, test.wantParentHash, blockResponse.Block.ParentID.Hash)

			assert.Equal(t, test.wantTimestamp, blockResponse.Block.Timestamp)

			if test.validateTransactions != nil {
				test.validateTransactions(blockResponse.Block.Transactions)
			}
		})
	}
}

func TestAPI_BlockHandlesErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		validBlockHash   = "810c9d25535107ba8729b1f26af2552e63d7b38b1e4cb8c848498faea1354cbd"
		validBlockHeight = 44

		trimmedBlockHash = "dab186b45199c0c26060ea09288b2f16032da40fc54c81bb2a8267a5c13906e" // blockID a character short
		lastHeight       = 425
	)

	var validBlockID = identifier.Block{
		Index: validBlockHeight,
		Hash:  validBlockHash,
	}

	tests := []struct {
		name string

		request rosetta.BlockRequest

		// HTTP/handler errors
		wantStatusCode int

		// rosetta errors - validated separately since it makes reporting mismatches more manageable
		wantRosettaError            meta.ErrorDefinition
		wantRosettaErrorDescription string
		wantRosettaErrorDetails     map[string]interface{}
	}{
		{
			name: "missing blockchain name",
			request: rosetta.BlockRequest{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
				BlockID: validBlockID,
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: blockchain field is empty",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "invalid blockchain name",
			request: rosetta.BlockRequest{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
				BlockID: validBlockID,
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier blockchain (have: %s, want: %s)", invalidBlockchain, dps.FlowBlockchain),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": invalidBlockchain, "network": dps.FlowTestnet.String()},
		},
		{
			name: "missing network name",
			request: rosetta.BlockRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
				BlockID: validBlockID,
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: network field is empty",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "invalid network name",
			request: rosetta.BlockRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    invalidNetwork,
				},
				BlockID: validBlockID,
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier network (have: %s, want: %s)", invalidNetwork, dps.FlowTestnet.String()),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": dps.FlowBlockchain, "network": invalidNetwork},
		},
		{
			name: "missing block height and hash",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: 0,
					Hash:  "",
				},
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "block identifier: at least one of hash or index is required",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "invalid length of block id",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: 43,
					Hash:  trimmedBlockHash,
				},
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: fmt.Sprintf("block identifier: hash field has wrong length (have: %d, want: %d)", len(trimmedBlockHash), validBlockHashLen),
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "missing block height",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Hash: validBlockHash,
				},
			},

			wantStatusCode:              http.StatusInternalServerError,
			wantRosettaError:            configuration.ErrorInternal,
			wantRosettaErrorDescription: "could not validate block: block access with hash currently not supported",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "invalid block hash",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: 13,
					Hash:  invalidBlockHash,
				},
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidBlock,
			wantRosettaErrorDescription: "block hash is not a valid hex-encoded string",
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(13), "hash": invalidBlockHash},
		},
		{
			name: "unknown block",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: lastHeight + 1,
				},
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorUnknownBlock,
			wantRosettaErrorDescription: fmt.Sprintf("block index is above last indexed block (last: %d)", lastHeight),
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(426), "hash": ""},
		},
		{
			name: "mismatched block height and hash",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: validBlockHeight - 1,
					Hash:  validBlockHash,
				},
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidBlock,
			wantRosettaErrorDescription: fmt.Sprintf("block hash does not match known hash for height (known: %s)", knownHeaders(validBlockHeight-1).ID().String()),
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(validBlockHeight - 1), "hash": validBlockHash},
		},
		{
			// effectively the same as the 'missing blockchain name' test case, since it's the first check we'll do
			name:    "empty block request",
			request: rosetta.BlockRequest{},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: blockchain field is empty",
			wantRosettaErrorDetails:     nil,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			enc, err := json.Marshal(test.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/block", bytes.NewReader(enc))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			rec := httptest.NewRecorder()
			ctx := echo.New().NewContext(req, rec)

			// execute the request
			err = api.Block(ctx)
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

func TestAPI_BlockHandlesMalformedRequest(t *testing.T) {

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
			"network_identifier": {
				"blockchain": "flow",
				"network": "flow-testnet"
			},
			"block_identifier": {
				"index": 13,
				"hash": "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"
			}`

		validJSON = `
		{
			"network_identifier": {
				"blockchain": "flow",
				"network": "flow-testnet"
			},
			"block_identifier": {
				"index": 13,
				"hash": "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"
			}
		}`
	)

	tests := []struct {
		name     string
		payload  []byte
		mimeType string
	}{
		{
			name:     "empty request",
			payload:  []byte(``),
			mimeType: echo.MIMEApplicationJSON,
		},
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

			req := httptest.NewRequest(http.MethodPost, "/block", bytes.NewReader(test.payload))
			req.Header.Set(echo.HeaderContentType, test.mimeType)

			rec := httptest.NewRecorder()
			ctx := echo.New().NewContext(req, rec)

			err := api.Block(ctx)
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

// blockRequest generates a BlockRequest with the specified parameters.
func blockRequest(header flow.Header) rosetta.BlockRequest {

	return rosetta.BlockRequest{
		NetworkID: defaultNetwork(),
		BlockID: identifier.Block{
			Index: header.Height,
			Hash:  header.ID().String(),
		},
	}
}

func validateTransfer(t *testing.T, hash string, from string, to string, amount int64) validateTxFunc {

	t.Helper()

	return func(transactions []*object.Transaction) {

		require.Len(t, transactions, 1)

		tx := transactions[0]

		assert.Equal(t, tx.ID.Hash, hash)
		assert.Equal(t, len(tx.Operations), 2)

		// operations come in pairs
		// - one is a negative transfer of funds (for the sender) and another one is a positive one (for the receiver)

		require.Equal(t, len(tx.Operations), 2)

		op1 := tx.Operations[0]
		op2 := tx.Operations[1]

		// verify the first operation data

		// validate operation and status
		assert.Equal(t, op1.Type, dps.OperationTransfer)
		assert.Equal(t, op1.Status, dps.StatusCompleted)

		// validate currency
		assert.Equal(t, op1.Amount.Currency.Symbol, dps.FlowSymbol)
		assert.Equal(t, op1.Amount.Currency.Decimals, uint(dps.FlowDecimals))

		// validate address
		address := op1.AccountID.Address
		if address != from && address != to {
			t.Errorf("unexpected account address (%v)", address)
		}

		// validate transfered amount
		wantValue := strconv.FormatInt(amount, 10)
		if address == from {
			wantValue = "-" + wantValue
		}

		assert.Equal(t, op1.Amount.Value, wantValue)

		// validate related operation is op2
		if assert.Len(t, op1.RelatedIDs, 1) {
			assert.Equal(t, op1.RelatedIDs[0], op2.ID)
		}

		// verify the second operation

		// validate operation and status
		assert.Equal(t, op2.Type, dps.OperationTransfer)
		assert.Equal(t, op2.Status, dps.StatusCompleted)

		// validate currency
		assert.Equal(t, op2.Amount.Currency.Symbol, dps.FlowSymbol)
		assert.Equal(t, op2.Amount.Currency.Decimals, uint(dps.FlowDecimals))

		// validate address
		address = op2.AccountID.Address
		if address != from && address != to {
			t.Errorf("unexpected account address (%v)", address)
		}

		// validate transfered amount
		wantValue = strconv.FormatInt(amount, 10)
		if address == from {
			wantValue = "-" + wantValue
		}

		assert.Equal(t, op2.Amount.Value, wantValue)

		// validate related operation is op1
		if assert.Len(t, op2.RelatedIDs, 1) {
			assert.Equal(t, op2.RelatedIDs[0], op1.ID)
		}
	}
}
