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
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/fail"
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

func (d *Data) Block(ctx echo.Context) error {

	var req BlockRequest
	err := ctx.Bind(&req)
	if err != nil {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("could not unmarshal request", fail.WithError(err)))
	}

	if req.NetworkID.Blockchain == "" {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("blockchain identifier: blockchain field is empty"))
	}
	if req.NetworkID.Network == "" {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("blockchain identifier: network field is empty"))
	}

	if req.BlockID.Index == 0 && req.BlockID.Hash == "" {
		return httpError(http.StatusBadRequest, fail.InvalidFormat("block identifier: at least one of hash or index is required"))
	}
	if req.BlockID.Hash != "" && len(req.BlockID.Hash) != hexIDSize {
		return httpError(
			http.StatusBadRequest,
			fail.InvalidFormat("block identifier: hash field has wrong length",
				fail.WithInt("have_length", len(req.BlockID.Hash)),
				fail.WithInt("want_length", hexIDSize),
			))
	}

	err = d.config.Check(req.NetworkID)
	var netErr fail.InvalidNetwork
	if errors.As(err, &netErr) {
		return httpError(http.StatusUnprocessableEntity, netErr.RosettaError())
	}
	if err != nil {
		return httpError(http.StatusInternalServerError, fail.Internal("could not validate network", fail.WithError(err)))
	}

	block, other, err := d.retrieve.Block(req.BlockID)

	var ibErr fail.InvalidBlock
	if errors.As(err, &ibErr) {
		return httpError(http.StatusUnprocessableEntity, ibErr.RosettaError())
	}
	var ubErr fail.UnknownBlock
	if errors.As(err, &ubErr) {
		return httpError(http.StatusUnprocessableEntity, ubErr.RosettaError())
	}

	if err != nil {
		return httpError(http.StatusInternalServerError, fail.Internal("could not retrieve block", fail.WithError(err)))
	}

	res := BlockResponse{
		Block:             block,
		OtherTransactions: other,
	}

	return ctx.JSON(http.StatusOK, res)
}
