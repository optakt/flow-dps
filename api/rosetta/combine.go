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

	"github.com/optakt/flow-dps/rosetta/request"
	"github.com/optakt/flow-dps/rosetta/response"
)

// Combine implements the /construction/combine endpoint of the Rosetta Construction API.
// It creates a signed transaction by combining an unsigned transaction and
// a list of signatures.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#constructioncombine
func (c *Construction) Combine(ctx echo.Context) error {

	var req request.Combine
	err := ctx.Bind(&req)
	if err != nil {
		return unpackError(err)
	}

	err = c.validate.Request(req)
	if err != nil {
		return formatError(err)
	}

	signed, err := c.transact.AttachSignatures(req.UnsignedTransaction, req.Signatures)
	if err != nil {
		return apiError(txSigning, err)
	}

	res := response.Combine{
		SignedTransaction: signed,
	}

	return ctx.JSON(statusOK, res)
}
