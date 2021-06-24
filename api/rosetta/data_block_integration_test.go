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
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type validateBlockFunc func(identifier.Block)
type validateTxFunc func([]*object.Transaction)

func TestAPI_Block(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	// Headers of known blocks to verify.
	var (
		firstHeader = knownHeader(1)
		midHeader1  = knownHeader(13)
		midHeader2  = knownHeader(43)
		midHeader3  = knownHeader(44)
		lastHeader  = knownHeader(425) // Header of last indexed block
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
			// Initial transfer of currency from the root account to the user - 100 tokens.
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
			// Transaction between two users.
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
			validateBlock:        validateBlock(t, midHeader3.Height, midHeader3.ID().String()), // Verify that the returned block ID has both height and hash
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

			rec, ctx, err := setupRecorder(blockEndpoint, test.request)
			require.NoError(t, err)

			err = api.Block(ctx)
			assert.NoError(t, err)

			// Unpack response.
			var blockResponse rosetta.BlockResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &blockResponse))
			require.NotNil(t, blockResponse.Block)

			// Validate index/hash of returned block.
			test.validateBlock(blockResponse.Block.ID)

			// Verify the index/hash of the parent block.
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

		trimmedBlockHash = "dab186b45199c0c26060ea09288b2f16032da40fc54c81bb2a8267a5c13906e" // BlockID a character too short
		lastHeight       = 425
	)

	var validBlockID = identifier.Block{
		Index: validBlockHeight,
		Hash:  validBlockHash,
	}

	tests := []struct {
		name string

		request rosetta.BlockRequest

		checkErr assert.ErrorAssertionFunc
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

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
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

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
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

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
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

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
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

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
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

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "missing block height",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Hash: validBlockHash,
				},
			},

			checkErr: checkRosettaError(http.StatusInternalServerError, configuration.ErrorInternal),
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

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidBlock),
		},
		{
			name: "unknown block",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: lastHeight + 1,
				},
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorUnknownBlock),
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

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidBlock),
		},
		{
			// Effectively the same as the 'missing blockchain name' test case, since it's the first validation step.
			name:    "empty block request",
			request: rosetta.BlockRequest{},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			_, ctx, err := setupRecorder(blockEndpoint, test.request)
			require.NoError(t, err)

			// Execute the request.
			err = api.Block(ctx)
			test.checkErr(t, err)
		})
	}
}

func TestAPI_BlockHandlesMalformedRequest(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		// Network field is an integer instead of a string.
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

			_, ctx, err := setupRecorder(blockEndpoint, test.payload, test.prepare)
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

		// Operations come in pairs. A negative transfer of funds for the sender and a positive one for the receiver.
		require.Equal(t, len(tx.Operations), 2)

		op1 := tx.Operations[0]
		op2 := tx.Operations[1]

		// Validate operation and status.
		assert.Equal(t, op1.Type, dps.OperationTransfer)
		assert.Equal(t, op1.Status, dps.StatusCompleted)

		// Validate currency.
		assert.Equal(t, op1.Amount.Currency.Symbol, dps.FlowSymbol)
		assert.Equal(t, op1.Amount.Currency.Decimals, uint(dps.FlowDecimals))

		// Validate address.
		address := op1.AccountID.Address
		if address != from && address != to {
			t.Errorf("unexpected account address (%v)", address)
		}

		// Validate transferred amount.
		wantValue := strconv.FormatInt(amount, 10)
		if address == from {
			wantValue = "-" + wantValue
		}

		assert.Equal(t, op1.Amount.Value, wantValue)

		// Validate related operations.
		if assert.Len(t, op1.RelatedIDs, 1) {
			assert.Equal(t, op1.RelatedIDs[0], op2.ID)
		}

		// Validate operation and status.
		assert.Equal(t, op2.Type, dps.OperationTransfer)
		assert.Equal(t, op2.Status, dps.StatusCompleted)

		// Validate currency.
		assert.Equal(t, op2.Amount.Currency.Symbol, dps.FlowSymbol)
		assert.Equal(t, op2.Amount.Currency.Decimals, uint(dps.FlowDecimals))

		// Validate address.
		address = op2.AccountID.Address
		if address != from && address != to {
			t.Errorf("unexpected account address (%v)", address)
		}

		// Validate transferred amount.
		wantValue = strconv.FormatInt(amount, 10)
		if address == from {
			wantValue = "-" + wantValue
		}

		assert.Equal(t, op2.Amount.Value, wantValue)

		// Validate related operations.
		if assert.Len(t, op2.RelatedIDs, 1) {
			assert.Equal(t, op2.RelatedIDs[0], op1.ID)
		}
	}
}
