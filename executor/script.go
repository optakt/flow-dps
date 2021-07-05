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

package executor

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/onflow/cadence"

	"github.com/optakt/flow-dps/models/convert"
)

// ScriptRequest describes the input data needed for script execution.
type ScriptRequest struct {
	Height    *uint64  `json:"height"`
	Script    string   `json:"script"`
	Arguments []string `json:"arguments"`
}

// ScriptResponse describes the output data of the script execution.
// The response includes all of the input arguments, as well as the
// execution result.
type ScriptResponse struct {
	Height    uint64          `json:"height"`
	Script    string          `json:"script"`
	Arguments []string        `json:"arguments,omitempty"`
	Result    json.RawMessage `json:"result"`
}

// Script endpoint handles execution of Cadence scripts at arbitrary heights.
func (e *Executor) Script(ctx echo.Context) error {

	// Unpack request data.
	var script ScriptRequest
	err := ctx.Bind(&script)
	if err != nil {
		return newHTTPError(http.StatusBadRequest, "could not unmarshal request", err)
	}

	// Validate mandatory arguments - height and script text are required.
	if script.Height == nil || script.Script == "" {
		return newHTTPError(http.StatusBadRequest, "missing height or script text", nil)
	}

	// Convert the script arguments.
	args := make([]cadence.Value, 0, len(script.Arguments))
	for _, arg := range script.Arguments {
		val, err := convert.ParseCadenceArgument(arg)
		if err != nil {
			return newHTTPError(http.StatusBadRequest, "could not parse cadence value", err)
		}

		args = append(args, val)
	}

	// Execute the script.
	res, err := e.invoker.Script(*script.Height, []byte(script.Script), args)
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "could not execute script", err)
	}

	// Prepare and send the response.

	payload, err := json.Marshal(res)
	if err != nil {
		return newHTTPError(http.StatusInternalServerError, "could not marshal response", err)
	}

	out := ScriptResponse{
		Height:    *script.Height,
		Script:    script.Script,
		Arguments: script.Arguments,
		Result:    payload,
	}

	return ctx.JSON(http.StatusOK, out)
}
