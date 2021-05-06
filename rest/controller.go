package rest

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	fledger "github.com/onflow/flow-go/ledger"

	"github.com/awfm9/flow-dps/ledger"
	"github.com/awfm9/flow-dps/model"
)

type Controller struct {
	core *ledger.Core
}

func NewController(core *ledger.Core) (*Controller, error) {
	c := &Controller{
		core: core,
	}
	return c, nil
}

func (c *Controller) GetRegister(ctx echo.Context) error {

	key, err := hex.DecodeString(ctx.Param("key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	raw := c.core.Raw()

	var height uint64
	heightParam := ctx.QueryParam("height")
	if heightParam != "" {
		height, err = strconv.ParseUint(heightParam, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		raw = raw.WithHeight(height)
	}

	value, err := raw.Get(key)
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

func (c *Controller) GetPayloads(ctx echo.Context) error {

	hash, err := hex.DecodeString(ctx.Param("hash"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	var keys []fledger.Key
	pathsParam := ctx.QueryParam("paths")
	if pathsParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	keysHex := strings.Split(pathsParam, ",")
	for _, keyHex := range keysHex {
		keyBytes, err := hex.DecodeString(keyHex)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		// TODO: find a proper way to encode keys / key parts with REST
		key := fledger.Key{
			KeyParts: []fledger.KeyPart{
				{Type: 0, Value: keyBytes},
			},
		}
		keys = append(keys, key)
	}

	// TODO: decide how to encode ledger keys
	query, err := fledger.NewQuery(hash, keys)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	ledger := c.core.Ledger()

	versionParam := ctx.QueryParam("version")
	if versionParam != "" {
		version, err := strconv.ParseUint(versionParam, 10, 8)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		ledger = ledger.WithVersion(uint8(version))
	}

	err = ctx.Bind(query)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	payloads, err := ledger.Get(query)
	if errors.Is(err, model.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, err)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return ctx.JSON(http.StatusOK, payloads)
}
