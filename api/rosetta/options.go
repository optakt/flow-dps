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

// Options implements the /network/options endpoint of the Rosetta Data API.
// See https://www.rosetta-api.org/docs/NetworkApi.html#networkoptions
func (d *Data) Options(ctx echo.Context) error {

	// Decode the network list request from the HTTP request JSON body.
	var req request.Options
	err := ctx.Bind(&req)
	if err != nil {
		return unpackError(err)
	}

	err = d.validate.Request(req)
	if err != nil {
		return formatError(err)
	}

	// Create the allow object, which is native to the response.
	allow := response.OptionsAllow{
		OperationStatuses:       d.config.Statuses(),
		OperationTypes:          d.config.Operations(),
		Errors:                  d.config.Errors(),
		HistoricalBalanceLookup: true,
		CallMethods:             []string{},
		BalanceExemptions:       []struct{}{},
		MempoolCoins:            false,
	}

	res := response.Options{
		Version: d.config.Version(),
		Allow:   allow,
	}

	return ctx.JSON(statusOK, res)
}
