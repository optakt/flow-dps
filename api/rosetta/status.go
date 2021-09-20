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

// Status implements the /network/status endpoint of the Rosetta Data API.
// See https://www.rosetta-api.org/docs/NetworkApi.html#networkstatus
func (d *Data) Status(ctx echo.Context) error {

	var req request.Status
	err := ctx.Bind(&req)
	if err != nil {
		return unpackError(err)
	}

	err = d.validate.Request(req)
	if err != nil {
		return formatError(err)
	}

	oldest, _, err := d.retrieve.Oldest()
	if err != nil {
		return apiError(oldestRetrieval, err)
	}

	current, timestamp, err := d.retrieve.Current()
	if err != nil {
		return apiError(currentRetrieval, err)
	}

	res := response.Status{
		CurrentBlockID:        current,
		CurrentBlockTimestamp: timestamp.UnixNano() / 1_000_000,
		OldestBlockID:         oldest,
		GenesisBlockID:        oldest,
		Peers:                 []struct{}{},
	}

	return ctx.JSON(statusOK, res)
}
