package rest

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"

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

func (c *Controller) GetValue(ctx echo.Context) error {

	keysParam := ctx.Param("keys")
	keysEncoded := strings.Split(keysParam, ":")
	if len(keysEncoded) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, errors.New("need at least one encoded ledger key"))
	}
	var keys []ledger.Key
	for _, keyEncoded := range keysEncoded {
		key, err := decodeKey(keyEncoded)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		keys = append(keys, key)
	}

	hash, err := hex.DecodeString(ctx.QueryParam("hash"))
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

	query, err := ledger.NewQuery(hash, keys)
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
