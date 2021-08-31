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

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

// BlockRequest implements the request schema for /block.
// See https://www.rosetta-api.org/docs/BlockApi.html#request
type BlockRequest struct {
	NetworkID identifier.Network `json:"network_identifier"`
	BlockID   identifier.Block   `json:"block_identifier"`
}

// BlockResponse implements the response schema for /block.
// See https://www.rosetta-api.org/docs/BlockApi.html#200---ok
type BlockResponse struct {
	Block             *object.Block            `json:"block"`
	OtherTransactions []identifier.Transaction `json:"other_transactions,omitempty"`
}

// Block implements the /block endpoint of the Rosetta Data API.
// See https://www.rosetta-api.org/docs/BlockApi.html#block
func (d *Data) Block(ctx echo.Context) error {

	var req BlockRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, invalidEncoding(invalidJSON, err))
	}

	if req.NetworkID.Blockchain == "" {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(blockchainEmpty))
	}
	if req.NetworkID.Network == "" {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(networkEmpty))
	}

	rosBlockID := req.BlockID
	// If both index and hash are not populated, it's assumed that the client is making
	// a request at the current block.
	if rosBlockID.Index == nil && rosBlockID.Hash == "" {
		current, _, err := d.retrieve.Current()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, internal(currentRetrieval, err))
		}
		rosBlockID = current
	}
	if rosBlockID.Hash != "" && len(rosBlockID.Hash) != hexIDSize {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(blockLength,
			withDetail("have_length", len(rosBlockID.Hash)),
			withDetail("want_length", hexIDSize),
		))
	}

	err = d.config.Check(req.NetworkID)
	var netErr failure.InvalidNetwork
	if errors.As(err, &netErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidNetwork(netErr))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, internal(networkCheck, err))
	}

	block, extraTxIDs, err := d.retrieve.Block(rosBlockID)

	var ibErr failure.InvalidBlock
	if errors.As(err, &ibErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, invalidBlock(ibErr))
	}
	var ubErr failure.UnknownBlock
	if errors.As(err, &ubErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, unknownBlock(ubErr))
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, internal(blockRetrieval, err))
	}

	res := BlockResponse{
		Block:             block,
		OtherTransactions: extraTxIDs,
	}

	return ctx.JSON(http.StatusOK, res)
}
