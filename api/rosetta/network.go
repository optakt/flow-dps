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
	"github.com/optakt/flow-dps/rosetta/object"
)

type NetworksRequest struct {
	Metadata map[string]string `json:"metadata"`
}

type NetworksResponse struct {
	NetworkIDs []identifier.Network `json:"network_identifiers"`
}

func (d *Data) Networks(ctx echo.Context) error {

	// Decode the network list request from the HTTP request JSON body.
	var req NetworksRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, object.AnyError(err))
	}

	// For now, we simply return a single network, which is the main chain of
	// the Flow blockchain network.
	network, err := d.retrieve.Network()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	res := NetworksResponse{
		NetworkIDs: []identifier.Network{network},
	}

	return ctx.JSON(http.StatusOK, res)
}
