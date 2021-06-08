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

package rosetta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/resource"
)

type BlockRequest struct {
	NetworkID identifier.Network `json:"network_identifier"`
	BlockID   identifier.Block   `json:"block_identifier"`
}

type BlockResponse struct {
	Block             *resource.Block          `json:"block"`
	OtherTransactions []identifier.Transaction `json:"other_transactions"`
}

// TODO: integration testing of Rosetta block endpoint
// => https://github.com/optakt/flow-dps/issues/47
func (d *Data) Block(ctx echo.Context) error {

	var req BlockRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat(err))
	}

	block, other, err := d.retrieve.Block(req.NetworkID, req.BlockID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(err))
	}

	res := BlockResponse{
		Block:             block,
		OtherTransactions: other,
	}

	return ctx.JSON(http.StatusOK, res)
}
