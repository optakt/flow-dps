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

// Transaction implements the /block/transaction endpoint of the Rosetta Data API.
// See https://www.rosetta-api.org/docs/BlockApi.html#blocktransaction
func (d *Data) Transaction(ctx echo.Context) error {

	var req request.Transaction
	err := ctx.Bind(&req)
	if err != nil {
		return unpackError(err)
	}

	err = d.validate.Request(req)
	if err != nil {
		return formatError(err)
	}

	err = d.validate.CompleteBlockID(req.BlockID)
	if err != nil {
		return formatError(err)
	}

	transaction, err := d.retrieve.Transaction(req.BlockID, req.TransactionID)
	if err != nil {
		return apiError(txRetrieval, err)
	}

	res := response.Transaction{
		Transaction: transaction,
	}

	return ctx.JSON(statusOK, res)
}
