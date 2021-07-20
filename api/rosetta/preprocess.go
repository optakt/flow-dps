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
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type PreprocessRequest struct {
	NetworkID  identifier.Network `json:"network_identifier"`
	Operations []object.Operation `json:"operations"`

	// TODO: check if we need max fee and suggested fee multiplier
}

type PreprocessResponse struct {
}

func (c *Construction) Preprocess(ctx echo.Context) error {

	var req PreprocessRequest
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

	if len(req.Operations) != 2 {
		return echo.NewHTTPError(http.StatusBadRequest, invalidFormat(txInvalidOpCount))
	}

	intent, err := c.parser.CreateTransfer(req.Operations)
	if err != nil {
		// TODO: change error handling
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("could not determine transaction intent: %w", err))
	}

	_ = intent

	return nil
}
