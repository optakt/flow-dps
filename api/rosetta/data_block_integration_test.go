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
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/meta"
	rosettaobj "github.com/optakt/flow-dps/rosetta/rosetta"
)

type blockIDValidationFn func(identifier.Block)
type transactionValidationFn func(*rosettaobj.Transaction)

func TestGetBlock(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	tests := []struct {
		name string

		request rosetta.BlockRequest

		wantTimestamp          int64
		wantParentHash         string
		transactionValidator   transactionValidationFn
		customBlockIDValidator blockIDValidationFn
	}{
		{
			name:           "child of first block",
			request:        blockRequest(1, knownBlockID(1)),
			wantTimestamp:  1621337323243,
			wantParentHash: knownBlockID(0),
		},
		{
			// initial transfer of currency from the root account to the user - 100 tokens
			name:                 "block mid-chain with transactions",
			request:              blockRequest(13, knownBlockID(13)),
			wantTimestamp:        1621338403243,
			wantParentHash:       knownBlockID(12),
			transactionValidator: validateSingleTransfer(t, "a9c9ab28ea76b7dbfd1f2666f74348e4188d67cf68248df6634cee3f06adf7b1", "8c5303eaa26202d6", "754aed9de6197641", 100_00000000),
		},
		{
			name:           "block mid-chain without transactions",
			request:        blockRequest(43, knownBlockID(43)),
			wantTimestamp:  1621341103243,
			wantParentHash: knownBlockID(42),
		},
		{
			// transaction between two users
			name:                 "second block mid-chain with transactions",
			request:              blockRequest(44, knownBlockID(44)),
			wantTimestamp:        1621341193243,
			wantParentHash:       knownBlockID(43),
			transactionValidator: validateSingleTransfer(t, "d5c18baf6c8d11f0693e71dbb951c4856d4f25a456f4d5285a75fd73af39161c", "754aed9de6197641", "631e88ae7f1d7c20", 1),
		},
		{
			// same as above, but verify lookup by index only works
			name: "lookup of a block mid-chain by index only",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetworkID(),
				BlockID:   identifier.Block{Index: 44}},
			wantTimestamp:          1621341193243,
			wantParentHash:         knownBlockID(43),
			transactionValidator:   validateSingleTransfer(t, "d5c18baf6c8d11f0693e71dbb951c4856d4f25a456f4d5285a75fd73af39161c", "754aed9de6197641", "631e88ae7f1d7c20", 1),
			customBlockIDValidator: validateBlockID(t, 44, knownBlockID(44)), // verify that the returned block ID has both height and hash
		},
		{
			name:           "last indexed block",
			request:        blockRequest(425, knownBlockID(425)),
			wantTimestamp:  1621375483243,
			wantParentHash: knownBlockID(424),
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

			if assert.NotNil(t, blockResponse.Block) {

				// verify block ID of the returned block either using the provided validator,
				// or by comparing the requested and returned block height and hash
				if test.customBlockIDValidator != nil {
					test.customBlockIDValidator(blockResponse.Block.ID)
				} else {
					assert.Equal(t, test.request.BlockID.Index, blockResponse.Block.ID.Index)
					assert.Equal(t, test.request.BlockID.Hash, blockResponse.Block.ID.Hash)
				}

				// verify the parent block index is correct
				assert.Equal(t, test.request.BlockID.Index-1, blockResponse.Block.ParentID.Index)
				assert.Equal(t, test.wantParentHash, blockResponse.Block.ParentID.Hash)

				assert.Equal(t, test.wantTimestamp, blockResponse.Block.Timestamp)

				if test.transactionValidator != nil {

					if assert.GreaterOrEqual(t, len(blockResponse.Block.Transactions), 1) {
						test.transactionValidator(blockResponse.Block.Transactions[0])
					}
				}
			}
		})
	}
}

func TestBlockErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		invalidBlockchainName = "not-flow"
		invalidNetworkName    = "not-flow-testnet"

		trimmedBlockId       = "dab186b45199c0c26060ea09288b2f16032da40fc54c81bb2a8267a5c13906e"  // blockID a character short
		invalidBlockHash     = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz" // invalid hex value
		validBlockIDLength   = 64
		lastKnownBlockHeight = 425
	)

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
			name: "missing network blockchain name",
			request: rosetta.BlockRequest{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
				BlockID: identifier.Block{
					Index: 44,
					Hash:  knownBlockID(44),
				},
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: blockchain field is empty",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "wrong network blockchain name",
			request: rosetta.BlockRequest{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchainName,
					Network:    dps.FlowTestnet.String(),
				},
				BlockID: identifier.Block{
					Index: 44,
					Hash:  knownBlockID(44),
				},
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier blockchain (have: %s, want: %s)", invalidBlockchainName, dps.FlowBlockchain),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": invalidBlockchainName, "network": dps.FlowTestnet.String()},
		},
		{
			name: "missing network name",
			request: rosetta.BlockRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
				BlockID: identifier.Block{
					Index: 44,
					Hash:  knownBlockID(44),
				},
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: "blockchain identifier: network field is empty",
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "wrong network name",
			request: rosetta.BlockRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    invalidNetworkName,
				},
				BlockID: identifier.Block{
					Index: 44,
					Hash:  knownBlockID(44),
				},
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidNetwork,
			wantRosettaErrorDescription: fmt.Sprintf("invalid network identifier network (have: %s, want: %s)", invalidNetworkName, dps.FlowTestnet.String()),
			wantRosettaErrorDetails:     map[string]interface{}{"blockchain": dps.FlowBlockchain, "network": invalidNetworkName},
		},
		{
			name: "missing block height and hash",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetworkID(),
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
			name: "wrong length of block id",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetworkID(),
				BlockID: identifier.Block{
					Index: 43,
					Hash:  trimmedBlockId,
				},
			},

			wantStatusCode:              http.StatusBadRequest,
			wantRosettaError:            configuration.ErrorInvalidFormat,
			wantRosettaErrorDescription: fmt.Sprintf("block identifier: hash field has wrong length (have: %d, want: %d)", len(trimmedBlockId), validBlockIDLength),
			wantRosettaErrorDetails:     nil,
		},
		{
			name: "missing block height",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetworkID(),
				BlockID: identifier.Block{
					Hash: knownBlockID(44),
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
				NetworkID: defaultNetworkID(),
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
				NetworkID: defaultNetworkID(),
				BlockID: identifier.Block{
					Index: lastKnownBlockHeight + 1,
				},
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorUnknownBlock,
			wantRosettaErrorDescription: fmt.Sprintf("block index is above last indexed block (last: %d)", lastKnownBlockHeight),
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(426), "hash": ""},
		},
		{
			name: "mismatched block height and hash",
			request: rosetta.BlockRequest{
				NetworkID: defaultNetworkID(),
				BlockID: identifier.Block{
					Index: 43,
					Hash:  knownBlockID(44),
				},
			},

			wantStatusCode:              http.StatusUnprocessableEntity,
			wantRosettaError:            configuration.ErrorInvalidBlock,
			wantRosettaErrorDescription: fmt.Sprintf("block hash does not match known hash for height (known: %s)", knownBlockID(43)),
			wantRosettaErrorDetails:     map[string]interface{}{"index": uint64(43), "hash": knownBlockID(44)},
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

func TestMalformedBlockRequest(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

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
			payload:  []byte(`{ "network_identifier": { "blockchain": "flow", "network": 99} }`),
			mimeType: echo.MIMEApplicationJSON,
		},
		{
			name: "unclosed bracket",
			payload: []byte(`{ "network_identifier": { "blockchain": "flow", "network": "flow-testnet" },
							   "block_identifier": { "index": 13, "hash": "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23" }`),
			mimeType: echo.MIMEApplicationJSON,
		},
		{
			// TODO: check if this should be treated as an error - echo will
			name: "valid payload with no mime type set",
			payload: []byte(`{ "network_identifier": { "blockchain": "flow", "network": "flow-testnet" },
							   "block_identifier": { "index": 13, "hash": "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23" } }`),
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
func blockRequest(height uint64, hash string) rosetta.BlockRequest {

	return rosetta.BlockRequest{
		NetworkID: defaultNetworkID(),
		BlockID: identifier.Block{
			Index: height,
			Hash:  hash,
		},
	}
}

func validateSingleTransfer(t *testing.T, hash string, from string, to string, amount int64) transactionValidationFn {

	t.Helper()

	return func(tx *rosettaobj.Transaction) {

		assert.Equal(t, tx.ID.Hash, hash)
		assert.Equal(t, len(tx.Operations), 2)

		relatedOperations := make(map[uint]uint)

		for _, op := range tx.Operations {

			// save related operation IDs in a map so we can cross reference them
			// TODO: check - is this special-casing the test too much perhaps?
			// operation can have multiple related operations,
			// but also this function does state that it's verifying a single transfer

			if assert.Len(t, op.RelatedIDs, 1) {
				relatedOperations[op.ID.Index] = op.RelatedIDs[0].Index
			}

			// validate operation and status
			assert.Equal(t, op.Type, dps.OperationTransfer)
			assert.Equal(t, op.Status, dps.StatusCompleted)

			// validate currency
			assert.Equal(t, op.Amount.Currency.Symbol, dps.FlowSymbol)
			assert.Equal(t, op.Amount.Currency.Decimals, uint(dps.FlowDecimals))

			// validate address
			address := op.AccountID.Address
			if address != from && address != to {
				t.Errorf("unexpected account address (%v)", address)
			}

			// validate transfered amount
			wantValue := strconv.FormatInt(amount, 10)
			if address == from {
				wantValue = "-" + wantValue
			}

			assert.Equal(t, op.Amount.Value, wantValue)
		}

		// cross-reference related operations - verify that the related operation backlinks to the original one
		for id, relatedID := range relatedOperations {
			assert.Contains(t, relatedOperations, relatedID)
			assert.Equal(t, id, relatedOperations[relatedID])
		}
	}
}

func validateBlockID(t *testing.T, height uint64, hash string) blockIDValidationFn {

	t.Helper()

	return func(blockID identifier.Block) {
		assert.Equal(t, height, blockID.Index)
		assert.Equal(t, hash, blockID.Hash)
	}
}

// blockID() looks like a natural function name, but don't want to occupy the variable name either
func knownBlockID(height uint64) string {

	// NOTE: map would be cleaner, but would be created on each call which seems wasteful

	switch height {

	case 0:
		return "d47b1bf7f37e192cf83d2bee3f6332b0d9b15c0aa7660d1e5322ea964667b333"
	case 1:
		return "9eac11ab78ebb9650803eea70a48399f772c64892823a051298d445459cdbc46"
	case 12:
		return "9035c558379b208eba11130c928537fe50ad93cdee314980fccb695aa31df7fc"
	case 13:
		return "af528bb047d6cd1400a326bb127d689607a096f5ccd81d8903dfebbac26afb23"
	case 42:
		return "91c00b22dc9b84281d293f6e1ff680133239addd8b0220a244554e1d96aed8e0"
	case 43:
		return "dab186b45199c0c26060ea09288b2f16032da40fc54c81bb2a8267a5c13906e6"
	case 44:
		return "810c9d25535107ba8729b1f26af2552e63d7b38b1e4cb8c848498faea1354cbd"
	case 424:
		return "6af26621eca92babda2df3ebcd2fe269946b3bf208183569258630e64486831d"
	case 425:
		return "594d59b2e61bb18b149ffaac2b27b0efe1854f6795cd3bb96a443c3676d78683"

	default:
		return ""
	}

}
