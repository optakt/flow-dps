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
	Peers                 []struct{}       `json:"peers"` // not used
}

// Status implements the /network/status endpoint of the Rosetta Data API.
// See https://www.rosetta-api.org/docs/NetworkApi.html#networkstatus
func (d *Data) Status(ctx echo.Context) error {

	var req StatusRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, invalidEncoding(invalidJSON, err))
	}

	err = d.validate.Request(req)
	if err != nil {
		return validationError(err)
	}

	err = d.config.Check(req.NetworkID)
	if err != nil {
		return echo.NewHTTPError(apiError(networkCheck, err))
	}

	oldest, _, err := d.retrieve.Oldest()
	if err != nil {
		return echo.NewHTTPError(apiError(oldestRetrieval, err))
	}

	current, timestamp, err := d.retrieve.Current()
	if err != nil {
		return echo.NewHTTPError(apiError(currentRetrieval, err))
	}

	res := StatusResponse{
		CurrentBlockID:        current,
		CurrentBlockTimestamp: timestamp.UnixNano() / 1_000_000,
		OldestBlockID:         oldest,
		GenesisBlockID:        oldest,
		Peers:                 []struct{}{},
	}

	return ctx.JSON(http.StatusOK, res)
}
