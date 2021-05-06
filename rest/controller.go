package rest

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/onflow/flow-go/ledger"

	"github.com/awfm9/flow-dps/model"
)

type Controller struct {
	state State
}

func NewController(state State) (*Controller, error) {
	c := &Controller{
		state: state,
	}
	return c, nil
}

func (c *Controller) GetRegister(ctx echo.Context) error {

	key, err := hex.DecodeString(ctx.Param("key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	state := c.state.Raw()

	var height uint64
	heightParam := ctx.QueryParam("height")
	if heightParam != "" {
		height, err = strconv.ParseUint(heightParam, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		state = state.WithHeight(height)
	}

	value, err := state.Get(key)
	if errors.Is(err, model.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, err)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	res := RegisterResponse{
		Height: height,
		Key:    key,
		Value:  value,
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
func (c *Controller) GetValue(ctx echo.Context) error {

	keys, err := DecodeKeys(ctx.Param("keys"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	_, _, commit := c.state.Latest()
	hashParam := ctx.QueryParam("hash")
	if hashParam != "" {
		hash, err := hex.DecodeString(hashParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		commit = hash
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

	query, err := ledger.NewQuery(commit, keys)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	err = ctx.Bind(query)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	values, err := state.Get(query)
	if errors.Is(err, model.ErrNotFound) {
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
