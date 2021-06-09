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
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

func TestGetBlock(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	tests := []struct {
		name string

		request rosetta.BlockRequest

		wantStatusCode       int
		wantTimestamp        int64 // TODO: init
		wantParentHash       string
		wantHandlerErr       assert.ErrorAssertionFunc
		transactionValidator transactionValidationFn
	}{
		{
			// TODO: think of a nice way to validate block responses

			// TODO: consider what to do here; it's a natural boundary element, but the parent block we will receive will be a bit weird (parent ID is uint64(-1))
			name:           "first block",
			request:        blockRequest(0, knownBlockID(0)),
			wantStatusCode: http.StatusOK,
			wantHandlerErr: assert.NoError,
			wantParentHash: "0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:           "child of first block",
			request:        blockRequest(1, knownBlockID(1)),
			wantStatusCode: http.StatusOK,
			wantHandlerErr: assert.NoError,
			wantParentHash: knownBlockID(0),
		},
		{
			// initial transfer of currency from the root account to the user
			name:                 "block mid-chain with transactions",
			request:              blockRequest(13, knownBlockID(13)),
			wantStatusCode:       http.StatusOK,
			wantHandlerErr:       assert.NoError,
			wantParentHash:       knownBlockID(12),
			transactionValidator: validateSingleTransfer(t, "a9c9ab28ea76b7dbfd1f2666f74348e4188d67cf68248df6634cee3f06adf7b1", "8c5303eaa26202d6", "754aed9de6197641", 100_00000000),
		},
		{
			name:           "block mid-chain without transactions",
			request:        blockRequest(43, knownBlockID(43)),
			wantStatusCode: http.StatusOK,
			wantHandlerErr: assert.NoError,
			wantParentHash: knownBlockID(42),
		},
		{
			// transaction between two users
			name:                 "second block mid-chain with transactions",
			request:              blockRequest(44, knownBlockID(44)),
			wantStatusCode:       http.StatusOK,
			wantHandlerErr:       assert.NoError,
			wantParentHash:       knownBlockID(43),
			transactionValidator: validateSingleTransfer(t, "d5c18baf6c8d11f0693e71dbb951c4856d4f25a456f4d5285a75fd73af39161c", "754aed9de6197641", "631e88ae7f1d7c20", 1),
		},
		{
			name:           "last indexed block",
			request:        blockRequest(425, knownBlockID(425)),
			wantStatusCode: http.StatusOK,
			wantHandlerErr: assert.NoError,
			wantParentHash: knownBlockID(424),
		},
		// TODO: add negative test cases
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
			test.wantHandlerErr(t, err)

			if test.wantStatusCode != http.StatusOK {
				e, ok := err.(*echo.HTTPError)
				require.True(t, ok)
				assert.Equal(t, test.wantStatusCode, e.Code)

				// nothing more to do, response validation should only be done for '200 OK' responses
				return
			}

			assert.Equal(t, test.wantStatusCode, rec.Result().StatusCode)

			// unpack response
			var blockResponse rosetta.BlockResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &blockResponse))

			// TODO: if we don't have 'other transactions' we can just require this
			if assert.NotNil(t, blockResponse.Block) {
				// verify that we got the data for the requested block
				assert.Equal(t, test.request.BlockID.Index, blockResponse.Block.ID.Index)
				assert.Equal(t, test.request.BlockID.Hash, blockResponse.Block.ID.Hash)

				// verify the parent block index is correct
				assert.Equal(t, test.request.BlockID.Index-1, blockResponse.Block.ParentID.Index)
				assert.Equal(t, test.wantParentHash, blockResponse.Block.ParentID.Hash)

				// assert.Equal(t, test.wantTimestamp, blockResponse.Block.Timestamp)

				if test.transactionValidator != nil {

					if assert.GreaterOrEqual(t, len(blockResponse.Block.Transactions), 1) {
						test.transactionValidator(blockResponse.Block.Transactions[0])
					}
				}
			}
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

type transactionValidationFn func(*object.Transaction)

// TODO: check test data - do we ever have a single block with multiple transactions
func validateSingleTransfer(t *testing.T, hash string, from string, to string, amount int64) transactionValidationFn {

	t.Helper()

	return func(tx *object.Transaction) {

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
