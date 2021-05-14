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

package rest

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/onflow/flow-go/ledger"

	"github.com/awfm9/flow-dps/models/dps"
)

type Controller struct {
	state dps.State
}

func NewController(state dps.State) (*Controller, error) {
	c := &Controller{
		state: state,
	}
	return c, nil
}

// TODO: integration testing of GetRegister endpoint
// => https://github.com/awfm9/flow-dps/issues/48
func (c *Controller) GetRegister(ctx echo.Context) error {

	key, err := hex.DecodeString(ctx.Param("key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	state := c.state.Raw()
	height := c.state.Last().Height()
	heightParam := ctx.QueryParam("height")
	if heightParam != "" {
		height, err = strconv.ParseUint(heightParam, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
	}
	state = state.WithHeight(height)

	value, err := state.Get(key)
	if errors.Is(err, dps.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, err)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	res := RegisterResponse{
		Height: height,
		Key:    hex.EncodeToString(key),
		Value:  hex.EncodeToString(value),
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetValue returns the payload value of an encoded Ledger entry in the same way
// as the Flow Ledger interface would. It takes an input that emulates the
// `ledger.Query` struct, in the following way:
// - The parameter `keys` is a semicolon (`:`) delimited set of `ledger.Key` strings.
// - Each `ledger.KeyPart` within the `ledger.Key` is delimited by a comma (`,`).
// - The type and the value of each `ledger.KeyPart` are delimited by a colon (`.`).
// - The value is encoded as a hexadecimal string.
// Additionally, the state hash and the pathfinder key version can be given as
// query parameters. If omitted, the state hash of the latest sealed block
// and the default pathfinder key encoding will be used.
// The response is returned as a simple array of hexadecimal strings.
// Example: GET /values/0.f647acg,4.ef67d11:0.f3321ab,3.ab321fe?hash=7ae6417ed5&version=1
// TODO: integration testing of GetValue endpoints
// => https://github.com/awfm9/flow-dps/issues/49
func (c *Controller) GetValue(ctx echo.Context) error {

	keys, err := DecodeKeys(ctx.Param("keys"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	state := c.state.Ledger()
	versionParam := ctx.QueryParam("version")
	if versionParam != "" {
		version, err := strconv.ParseUint(versionParam, 10, 8)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		state = state.WithVersion(uint8(version))
	}

	commit := c.state.Last().Commit()
	hashParam := ctx.QueryParam("hash")
	if hashParam != "" {
		commit, err = hex.DecodeString(hashParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
	}

	query, err := ledger.NewQuery(commit, keys)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	err = ctx.Bind(query)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	values, err := state.Get(query)
	if errors.Is(err, dps.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, err)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	valuesHex := make([]string, 0, len(values))
	for _, value := range values {
		valuesHex = append(valuesHex, hex.EncodeToString(value))
	}

	return ctx.JSON(http.StatusOK, valuesHex)
}
