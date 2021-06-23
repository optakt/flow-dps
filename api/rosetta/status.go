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
	errortype "errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/errors"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

type StatusRequest struct {
	NetworkID identifier.Network `json:"network_identifier"`
}

type StatusResponse struct {
	CurrentBlockID        identifier.Block `json:"current_block_identifier"`
	CurrentBlockTimestamp int64            `json:"current_block_timestamp"`
	OldestBlockID         identifier.Block `json:"oldest_block_identifier"`
}

func (d *Data) Status(ctx echo.Context) error {

	var req StatusRequest
	err := ctx.Bind(&req)
	if err != nil {
		return httpError(http.StatusBadRequest, errors.InvalidFormat("could not unmarshal request", errors.WithError(err)))
	}

	if req.NetworkID.Blockchain == "" {
		return httpError(http.StatusBadRequest, errors.InvalidFormat("blockchain identifier: blockchain field is missing"))
	}
	if req.NetworkID.Network == "" {
		return httpError(http.StatusBadRequest, errors.InvalidFormat("blockchain identifier: network field is missing"))
	}

	err = d.config.Check(req.NetworkID)
	var netErr errors.InvalidNetwork
	if errortype.As(err, &netErr) {
		return httpError(http.StatusUnprocessableEntity, netErr.RosettaError())
	}
	if err != nil {
		return httpError(http.StatusInternalServerError, errors.Internal("could not validate network", errors.WithError(err)))
	}

	oldest, _, err := d.retrieve.Oldest()
	if err != nil {
		return httpError(http.StatusInternalServerError, errors.Internal("could not retrieve oldest block", errors.WithError(err)))
	}
	current, timestamp, err := d.retrieve.Current()
	if err != nil {
		return httpError(http.StatusInternalServerError, errors.Internal("could not retrieve current block", errors.WithError(err)))
	}

	// TODO: See if it makes sense to include the genesis block information:
	// => https://github.com/optakt/flow-dps/issues/152

	res := StatusResponse{
		CurrentBlockID:        current,
		CurrentBlockTimestamp: timestamp.UnixNano() / 1_000_000,
		OldestBlockID:         oldest,
	}

	return ctx.JSON(http.StatusOK, res)
}
