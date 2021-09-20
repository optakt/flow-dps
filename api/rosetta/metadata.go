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

package rosetta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/object"
	"github.com/optakt/flow-dps/rosetta/request"
	"github.com/optakt/flow-dps/rosetta/response"
)

// Metadata implements the /construction/metadata endpoint of the Rosetta Construction API.
// Metadata endpoint returns information required for constructing the transaction.
// For Flow, that information includes the reference block and sequence number. Reference block
// is the last indexed block, and is used to track transaction expiration. Sequence number is
// the proposer account's public key sequence number. Sequence number is incremented for each
// transaction and is used to prevent replay attacks.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#constructionmetadata
func (c *Construction) Metadata(ctx echo.Context) error {

	var req request.Metadata
	err := ctx.Bind(&req)
	if err != nil {
		return unpackError(err)
	}

	err = c.validate.Request(req)
	if err != nil {
		return formatError(err)
	}

	current, _, err := c.retrieve.Current()
	if err != nil {
		return apiError(referenceBlockRetrieval, err)
	}

	sequence, err := c.retrieve.Sequence(current, req.Options.AccountID, 0)
	if err != nil {
		return apiError(sequenceNumberRetrieval, err)
	}

	// In the `parse` endpoint, we parse a transaction to produce the original metadata (and operations).
	// Since we can only deduce the block hash from the transaction, we will omit the block height from
	// the identifier here, to keep the data identical.
	res := response.Metadata{
		Metadata: object.Metadata{
			CurrentBlockID: current,
			SequenceNumber: sequence,
		},
	}

	return ctx.JSON(http.StatusOK, res)
}
