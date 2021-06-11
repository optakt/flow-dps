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
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type BlockRequest struct {
	NetworkID identifier.Network `json:"network_identifier"`
	BlockID   identifier.Block   `json:"block_identifier"`
}

type BlockResponse struct {
	Block             *object.Block            `json:"block"`
	OtherTransactions []identifier.Transaction `json:"other_transactions,omitempty"`
}

// TODO: integration testing of Rosetta block endpoint
// => https://github.com/optakt/flow-dps/issues/47
func (d *Data) Block(ctx echo.Context) error {

	var req BlockRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat(err.Error()))
	}

	if req.NetworkID.Blockchain == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("blockchain identifier: blockchain field is empty"))
	}
	if req.NetworkID.Network == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("blockchain identifier: network field is empty"))
	}

	if req.BlockID.Index == 0 && req.BlockID.Hash == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("block identifier: at least one of hash or index is required"))
	}
	if req.BlockID.Hash != "" && len(req.BlockID.Hash) != hexIDSize {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat("block identifier: hash field has wrong length (have: %d, want: %d)", len(req.BlockID.Hash), hexIDSize))
	}

	err = d.config.Check(req.NetworkID)
	var netErr failure.InvalidNetwork
	if errors.As(err, &netErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidNetwork(netErr))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(err))
	}

	block, other, err := d.retrieve.Block(req.BlockID)

	var ibErr failure.InvalidBlock
	if errors.As(err, &ibErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidBlock(ibErr))
	}
	var ubErr failure.UnknownBlock
	if errors.As(err, &ubErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, UnknownBlock(ubErr))
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(err))
	}

	res := BlockResponse{
		Block:             block,
		OtherTransactions: other,
	}

	return ctx.JSON(http.StatusOK, res)
}
