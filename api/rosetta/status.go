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
)

// StatusRequest implements the request schema for /network/status.
// See https://www.rosetta-api.org/docs/NetworkApi.html#request-2
type StatusRequest struct {
	NetworkID identifier.Network `json:"network_identifier"`
}

// StatusResponse implements the successful response schema for /network/status.
// See https://www.rosetta-api.org/docs/NetworkApi.html#200---ok-2
type StatusResponse struct {
	CurrentBlockID        identifier.Block `json:"current_block_identifier"`
	CurrentBlockTimestamp int64            `json:"current_block_timestamp"`
	OldestBlockID         identifier.Block `json:"oldest_block_identifier"`
	GenesisBlockID        identifier.Block `json:"genesis_block_identifier"`
}

// Status implements the /network/status endpoint of the Rosetta Data API.
// See https://www.rosetta-api.org/docs/NetworkApi.html#networkstatus
func (d *Data) Status(ctx echo.Context) error {

	var req StatusRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidEncoding(invalidJSON, err))
	}

	if req.NetworkID.Blockchain == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat(blockchainEmpty))
	}
	if req.NetworkID.Network == "" {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat(networkEmpty))
	}

	err = d.config.Check(req.NetworkID)
	var netErr failure.InvalidNetwork
	if errors.As(err, &netErr) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, InvalidNetwork(netErr))
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(networkCheck, err))
	}

	oldest, _, err := d.retrieve.Oldest()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(oldestRetrieval, err))
	}
	current, timestamp, err := d.retrieve.Current()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, Internal(currentRetrieval, err))
	}

	res := StatusResponse{
		CurrentBlockID:        current,
		CurrentBlockTimestamp: timestamp.UnixNano() / 1_000_000,
		OldestBlockID:         oldest,
		GenesisBlockID:        oldest,
	}

	// TODO: Implement genesis block return
	//       https://github.com/optakt/flow-dps/issues/229

	return ctx.JSON(http.StatusOK, res)
}
