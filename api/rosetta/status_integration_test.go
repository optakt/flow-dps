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

//go:build integration
// +build integration

package rosetta_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/request"
	"github.com/optakt/flow-dps/rosetta/response"
)

func TestAPI_Status(t *testing.T) {
	// TODO: Repair integration tests
	//       See https://github.com/optakt/flow-dps/issues/333
	t.Skip("integration tests disabled until new snapshot is generated")

	db := setupDB(t)
	api := setupAPI(t, db)

	oldestBlockID := knownHeader(0).ID().String()
	lastBlock := knownHeader(425)

	request := request.Status{
		NetworkID: defaultNetwork(),
	}

	rec, ctx, err := setupRecorder(statusEndpoint, request)
	require.NoError(t, err)

	err = api.Status(ctx)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

	var status response.Status
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &status))

	currentHeight := status.CurrentBlockID.Index
	require.NotNil(t, currentHeight)
	assert.Equal(t, *currentHeight, lastBlock.Height)

	assert.Equal(t, status.CurrentBlockID.Hash, lastBlock.ID().String())
	assert.Equal(t, status.CurrentBlockTimestamp, convert.RosettaTime(lastBlock.Timestamp))

	assert.Equal(t, status.OldestBlockID.Hash, oldestBlockID)

	oldestHeight := status.OldestBlockID.Index
	require.NotNil(t, oldestHeight)
	assert.Equal(t, *oldestHeight, uint64(0))

	assert.Equal(t, status.GenesisBlockID.Hash, oldestBlockID)

	genesisBlockHeight := status.GenesisBlockID.Index
	require.NotNil(t, genesisBlockHeight)
	assert.Equal(t, *genesisBlockHeight, uint64(0))
}

func TestAPI_StatusHandlesErrors(t *testing.T) {
	// TODO: Repair integration tests
	//       See https://github.com/optakt/flow-dps/issues/333
	t.Skip("integration tests disabled until new snapshot is generated")

	db := setupDB(t)
	api := setupAPI(t, db)

	tests := []struct {
		name string

		request request.Status

		checkError assert.ErrorAssertionFunc
	}{
		{
			name: "missing blockchain",
			request: request.Status{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid blockchain",
			request: request.Status{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
		{
			name: "missing network",
			request: request.Status{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid network",
			request: request.Status{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    invalidNetwork,
				},
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			_, ctx, err := setupRecorder(statusEndpoint, test.request)
			require.NoError(t, err)

			err = api.Status(ctx)
			test.checkError(t, err)
		})
	}
}

func TestAPI_StatusHandlerMalformedRequest(t *testing.T) {
	// TODO: Repair integration tests
	//       See https://github.com/optakt/flow-dps/issues/333
	t.Skip("integration tests disabled until new snapshot is generated")

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		// wrong type for 'network' field
		wrongFieldType = `{
			"network_identifier": {
				"blockchain": "flow",
				"network": 99
			}
		}`

		// malformed JSON - unclosed bracket
		unclosedBracket = `{
			"network_identifier": {
				"blockchain": "flow",
				"network": "flow-testnet"
			}`

		validPayload = `{
			"network_identifier": {
				"blockchain": "flow",
				"network": "flow-testnet"
			}
		}`
	)

	tests := []struct {
		name string

		payload []byte

		prepare func(*http.Request)
	}{
		{
			name:    "invalid status input types",
			payload: []byte(wrongFieldType),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			},
		},
		{
			name:    "invalid status json format",
			payload: []byte(unclosedBracket),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			},
		},
		{
			name:    "valid status payload with no MIME type",
			payload: []byte(validPayload),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, "")
			},
		},
	}

	for _, test := range tests {

		test := test
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			_, ctx, err := setupRecorder(statusEndpoint, test.payload, test.prepare)
			require.NoError(t, err)

			// execute the request
			err = api.Status(ctx)
			assert.Error(t, err)

			echoErr, ok := err.(*echo.HTTPError)
			require.True(t, ok)

			assert.Equal(t, http.StatusBadRequest, echoErr.Code)
			gotErr, ok := echoErr.Message.(rosetta.Error)
			require.True(t, ok)

			assert.Equal(t, configuration.ErrorInvalidEncoding, gotErr.ErrorDefinition)
		})
	}
}
