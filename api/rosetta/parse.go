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
	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/object"
	"github.com/optakt/flow-dps/rosetta/request"
	"github.com/optakt/flow-dps/rosetta/response"
)

// Parse implements the /construction/parse endpoint of the Rosetta Construction API.
// Parse endpoint parses both signed and unsigned transactions to understand the
// transaction's intent. Endpoint returns the list of operations, any relevant metadata,
// and, in the case of signed transaction, the list of signers.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#constructionparse
func (c *Construction) Parse(ctx echo.Context) error {

	var req request.Parse
	err := ctx.Bind(&req)
	if err != nil {
		return unpackError(err)
	}

	err = c.validate.Request(req)
	if err != nil {
		return formatError(err)
	}

	parse, err := c.transact.Parse(req.Transaction)
	if err != nil {
		return apiError(txParsing, err)
	}

	refBlockID, err := parse.BlockID()
	if err != nil {
		return apiError(txParsing, err)
	}

	signers, err := parse.Signers()
	if err != nil {
		return apiError(txParsing, err)
	}

	operations, err := parse.Operations()
	if err != nil {
		return apiError(txParsing, err)
	}

	sequence := parse.Sequence()
	metadata := object.Metadata{
		CurrentBlockID: refBlockID,
		SequenceNumber: sequence,
	}

	res := response.Parse{
		Operations: operations,
		SignerIDs:  signers,
		Metadata:   metadata,
	}

	return ctx.JSON(statusOK, res)
}
