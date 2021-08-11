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
		firstHeader  = knownHeader(0)
		secondHeader = knownHeader(1)
		midHeader1   = knownHeader(13)
		midHeader2   = knownHeader(43)
		midHeader3   = knownHeader(44)
		lastHeader   = knownHeader(425) // header of last indexed block
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
		wantParentHeight     uint64
		validateTransactions validateTxFunc
		validateBlock        validateBlockFunc
	}{
		{
			// First block. Besides the standard validation, it's also a special case
			// since according to the Rosetta spec, it should point to itself as the parent.
			name:    "first block",
			request: blockRequest(firstHeader),

			wantTimestamp:    convert.RosettaTime(firstHeader.Timestamp),
			wantParentHash:   firstHeader.ID().String(),
			wantParentHeight: firstHeader.Height,
			validateBlock:    validateByHeader(t, firstHeader),
		},
		{
			name:    "child of first block",
			request: blockRequest(secondHeader),

			wantTimestamp:    convert.RosettaTime(secondHeader.Timestamp),
			wantParentHash:   secondHeader.ParentID.String(),
			wantParentHeight: secondHeader.Height - 1,
			validateBlock:    validateByHeader(t, secondHeader),
		},
		{
			// Initial transfer of currency from the root account to the user - 100 tokens.
			name:    "block mid-chain with transactions",
			request: blockRequest(midHeader1),

			wantTimestamp:        convert.RosettaTime(midHeader1.Timestamp),
			wantParentHash:       midHeader1.ParentID.String(),
			wantParentHeight:     midHeader1.Height - 1,
			validateBlock:        validateByHeader(t, midHeader1),
			validateTransactions: validateTransfer(t, initialLoadTx, rootAccount, senderAccount, 100_00000000),
		},
		{
			name:    "block mid-chain without transactions",
			request: blockRequest(midHeader2),

			wantTimestamp:    convert.RosettaTime(midHeader2.Timestamp),
			wantParentHash:   midHeader2.ParentID.String(),
			wantParentHeight: midHeader2.Height - 1,
			validateBlock:    validateByHeader(t, midHeader2),
		},
		{
			// Transaction between two users.
			name:    "second block mid-chain with transactions",
			request: blockRequest(midHeader3),

			wantTimestamp:        convert.RosettaTime(midHeader3.Timestamp),
			wantParentHash:       midHeader3.ParentID.String(),
			wantParentHeight:     midHeader3.Height - 1,
			validateBlock:        validateByHeader(t, midHeader3),
			validateTransactions: validateTransfer(t, transferTx, senderAccount, receiverAccount, 1),
		},
		{
			name: "lookup of a block mid-chain by index only",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID:   identifier.Block{Index: &midHeader3.Height},
			},

			wantTimestamp:        convert.RosettaTime(midHeader3.Timestamp),
			wantParentHash:       midHeader3.ParentID.String(),
			wantParentHeight:     midHeader3.Height - 1,
			validateTransactions: validateTransfer(t, transferTx, senderAccount, receiverAccount, 1),
			validateBlock:        validateBlock(t, midHeader3.Height, midHeader3.ID().String()), // verify that the returned block ID has both height and hash
		},
		{
			name:    "last indexed block",
			request: blockRequest(lastHeader),

			wantTimestamp:    convert.RosettaTime(lastHeader.Timestamp),
			wantParentHash:   lastHeader.ParentID.String(),
			wantParentHeight: lastHeader.Height - 1,
			validateBlock:    validateByHeader(t, lastHeader),
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

			var blockResponse rosetta.BlockResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &blockResponse))
			require.NotNil(t, blockResponse.Block)

			test.validateBlock(blockResponse.Block.ID)

			assert.Equal(t, test.wantTimestamp, blockResponse.Block.Timestamp)

			// Verify that the information about the parent block (index and hash) is correct.
			assert.Equal(t, test.wantParentHash, blockResponse.Block.ParentID.Hash)

			if assert.NotNil(t, blockResponse.Block.ParentID.Index) {
				assert.Equal(t, test.wantParentHeight, *blockResponse.Block.ParentID.Index)
			}

			if test.validateTransactions != nil {
				test.validateTransactions(blockResponse.Block.Transactions)
			}
		})
	}
}

func TestAPI_BlockHandlesErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	var (
		validBlockHeight uint64 = 44
		lastHeight       uint64 = 425

		validBlockHash = knownHeader(validBlockHeight).ID().String()
	)

	const trimmedBlockHash = "dab186b45199c0c26060ea09288b2f16032da40fc54c81bb2a8267a5c13906e" // blockID a character too short

	var validBlockID = identifier.Block{
		Index: &validBlockHeight,
		Hash:  validBlockHash,
	}

	tests := []struct {
		name string

		request rosetta.BlockRequest

		checkErr assert.ErrorAssertionFunc
	}{
		{
			// Effectively the same as the 'missing blockchain name' test case, since it's the first validation step.
			name:    "empty block request",
			request: rosetta.BlockRequest{},

			checkErr: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
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
					Index: nil,
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
					Index: getUint64P(43),
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
					Index: getUint64P(13),
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
					Index: getUint64P(lastHeight + 1),
				},
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorUnknownBlock),
		},
		{
			name: "mismatched block height and hash",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetwork(),
				BlockID: identifier.Block{
					Index: getUint64P(validBlockHeight - 1),
					Hash:  validBlockHash,
				},
			},

			checkErr: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidBlock),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			_, ctx, err := setupRecorder(blockEndpoint, test.request)
			require.NoError(t, err)

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

			assert.Equal(t, configuration.ErrorInvalidEncoding, gotErr.ErrorDefinition)
			assert.NotEmpty(t, gotErr.Description)
		})
	}

}

// blockRequest generates a BlockRequest with the specified parameters.
func blockRequest(header flow.Header) rosetta.BlockRequest {

	return rosetta.BlockRequest{
		NetworkID: defaultNetwork(),
		BlockID: identifier.Block{
			Index: &header.Height,
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

		assert.Equal(t, op1.Type, dps.OperationTransfer)
		assert.Equal(t, op1.Status, dps.StatusCompleted)

		assert.Equal(t, op1.Amount.Currency.Symbol, dps.FlowSymbol)
		assert.Equal(t, op1.Amount.Currency.Decimals, uint(dps.FlowDecimals))

		address := op1.AccountID.Address
		if address != from && address != to {
			t.Errorf("unexpected account address (%v)", address)
		}

		wantValue := strconv.FormatInt(amount, 10)
		if address == from {
			wantValue = "-" + wantValue
		}

		assert.Equal(t, op1.Amount.Value, wantValue)

		assert.Equal(t, op2.Type, dps.OperationTransfer)
		assert.Equal(t, op2.Status, dps.StatusCompleted)

		assert.Equal(t, op2.Amount.Currency.Symbol, dps.FlowSymbol)
		assert.Equal(t, op2.Amount.Currency.Decimals, uint(dps.FlowDecimals))

		address = op2.AccountID.Address
		if address != from && address != to {
			t.Errorf("unexpected account address (%v)", address)
		}

		wantValue = strconv.FormatInt(amount, 10)
		if address == from {
			wantValue = "-" + wantValue
		}

		assert.Equal(t, op2.Amount.Value, wantValue)
	}
}
