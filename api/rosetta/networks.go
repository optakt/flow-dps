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
)

type NetworksRequest struct {
}

type NetworksResponse struct {
	NetworkIDs []identifier.Network `json:"network_identifiers"`
}

func (d *Data) Networks(ctx echo.Context) error {

	// Decode the network list request from the HTTP request JSON body.
	var req NetworksRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, InvalidFormat(err.Error()))
	}

	// Get the network we are running on from the configuration.
	res := NetworksResponse{
		NetworkIDs: []identifier.Network{d.config.Network()},
	}

	return ctx.JSON(http.StatusOK, res)
}
