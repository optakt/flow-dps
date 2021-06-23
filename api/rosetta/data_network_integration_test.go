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

// +build integration

package rosetta_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

func TestAPI_Status(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const (
		lastHeight    = 425
		oldestBlockID = "d47b1bf7f37e192cf83d2bee3f6332b0d9b15c0aa7660d1e5322ea964667b333"
	)

	lastBlock := knownHeader(425)

	request := rosetta.StatusRequest{
		NetworkID: defaultNetwork(),
	}

	rec, ctx, err := setupRecorder(statusEndpoint, request)
	require.NoError(t, err)

	err = api.Status(ctx)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

	// unpack response
	var status rosetta.StatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &status))

	// verify current block
	assert.Equal(t, status.CurrentBlockID.Index, lastBlock.Height)
	assert.Equal(t, status.CurrentBlockID.Hash, lastBlock.ID().String())
	assert.Equal(t, status.CurrentBlockTimestamp, convert.RosettaTime(lastBlock.Timestamp))

	// verify oldest block
	assert.Equal(t, status.OldestBlockID.Hash, oldestBlockID)

	// this is actually omitted from JSON, due to height being zero, and JSON having the 'omitempty' tag
	assert.Equal(t, status.OldestBlockID.Index, uint64(0))
}

func TestAPI_StatusHandlesErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	tests := []struct {
		name string

		request rosetta.StatusRequest

		checkError assert.ErrorAssertionFunc
	}{
		{
			name: "missing blockchain",
			request: rosetta.StatusRequest{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid blockchain",
			request: rosetta.StatusRequest{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
		{
			name: "missing network",
			request: rosetta.StatusRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid network",
			request: rosetta.StatusRequest{
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

func TestAPI_Networks(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	// network request is basically an empty payload at the moment,
	// there is a 'metadata' object that we're ignoring;
	// but we can have the scaffolding here in case something changes

	var netReq rosetta.NetworksRequest

	rec, ctx, err := setupRecorder(listEndpoint, netReq)
	require.NoError(t, err)

	// execute the request
	err = api.Networks(ctx)
	assert.NoError(t, err)

	var network rosetta.NetworksResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &network))

	if assert.Len(t, network.NetworkIDs, 1) {
		assert.Equal(t, network.NetworkIDs[0].Blockchain, dps.FlowBlockchain)
		assert.Equal(t, network.NetworkIDs[0].Network, dps.FlowTestnet.String())
	}
}

func TestAPI_Options(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	const errorCount = 9

	// verify version string is in the format of x.y.z
	versionRe := regexp.MustCompile(`\d+\.\d+\.\d+`)

	request := rosetta.OptionsRequest{
		NetworkID: defaultNetwork(),
	}

	rec, ctx, err := setupRecorder(optionsEndpoint, request)

	err = api.Options(ctx)
	require.NoError(t, err)

	// verify nominal case
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

	var options rosetta.OptionsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &options))

	// verify the version info
	// TODO: we're not currently doing exact match of the version, only of the format
	// check - should we?
	assert.Regexp(t, versionRe, options.Version.RosettaVersion)
	assert.Regexp(t, versionRe, options.Version.NodeVersion)
	assert.Regexp(t, versionRe, options.Version.MiddlewareVersion)

	assert.True(t, options.Allow.HistoricalBalanceLookup)

	if assert.Len(t, options.Allow.OperationStatuses, 1) {

		status := options.Allow.OperationStatuses[0]
		assert.Equal(t, status.Status, dps.StatusCompleted)
		assert.True(t, status.Successful)
	}

	if assert.Len(t, options.Allow.OperationTypes, 1) {
		assert.Equal(t, options.Allow.OperationTypes[0], dps.OperationTransfer)
	}

	if assert.Len(t, options.Allow.Errors, errorCount) {

		for i := uint(0); i < errorCount; i++ {

			rosettaErr := options.Allow.Errors[i]

			expectedCode := i + 1 // error codes start from 1

			assert.Equal(t, expectedCode, rosettaErr.Code)

			switch expectedCode {

			case 1:
				assert.Equal(t, configuration.ErrorInternal.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorInternal.Retriable, rosettaErr.Retriable)
			case 2:
				assert.Equal(t, configuration.ErrorInvalidFormat.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorInvalidFormat.Retriable, rosettaErr.Retriable)
			case 3:
				assert.Equal(t, configuration.ErrorInvalidNetwork.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorInvalidNetwork.Retriable, rosettaErr.Retriable)
			case 4:
				assert.Equal(t, configuration.ErrorInvalidAccount.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorInvalidAccount.Retriable, rosettaErr.Retriable)
			case 5:
				assert.Equal(t, configuration.ErrorInvalidCurrency.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorInvalidCurrency.Retriable, rosettaErr.Retriable)
			case 6:
				assert.Equal(t, configuration.ErrorInvalidBlock.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorInvalidBlock.Retriable, rosettaErr.Retriable)
			case 7:
				assert.Equal(t, configuration.ErrorInvalidTransaction.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorInvalidTransaction.Retriable, rosettaErr.Retriable)
			case 8:
				assert.Equal(t, configuration.ErrorUnknownBlock.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorUnknownBlock.Retriable, rosettaErr.Retriable)
			case 9:
				assert.Equal(t, configuration.ErrorUnknownCurrency.Message, rosettaErr.Message)
				assert.Equal(t, configuration.ErrorUnknownCurrency.Retriable, rosettaErr.Retriable)

			default:
				t.Errorf("unknown rosetta error received: (code: %v, message: '%v', retriable: %v", rosettaErr.Code, rosettaErr.Message, rosettaErr.Retriable)
			}
		}
	}

}

func TestAPI_OptionsHandlesErrors(t *testing.T) {

	db := setupDB(t)
	api := setupAPI(t, db)

	tests := []struct {
		name string

		request rosetta.OptionsRequest

		checkError assert.ErrorAssertionFunc
	}{
		{
			name: "missing blockchain",
			request: rosetta.OptionsRequest{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid blockchain",
			request: rosetta.OptionsRequest{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
		{
			name: "missing network",
			request: rosetta.OptionsRequest{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid network",
			request: rosetta.OptionsRequest{
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

			_, ctx, err := setupRecorder(optionsEndpoint, test.request)
			require.NoError(t, err)

			err = api.Options(ctx)
			test.checkError(t, err)
		})
	}

}

func TestAPI_StatusHandlerMalformedRequest(t *testing.T) {

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

			assert.Equal(t, configuration.ErrorInvalidFormat, gotErr.ErrorDefinition)
		})
	}
}

func TestAPI_OptionsHandlesMalformedRequest(t *testing.T) {

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
			name:    "invalid options input types",
			payload: []byte(wrongFieldType),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			},
		},
		{
			name:    "invalid options json format",
			payload: []byte(unclosedBracket),
			prepare: func(req *http.Request) {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			},
		},
		{
			name:    "valid options payload with no MIME type",
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

			_, ctx, err := setupRecorder(optionsEndpoint, test.payload, test.prepare)
			require.NoError(t, err)

			// execute the request
			err = api.Options(ctx)
			assert.Error(t, err)

			echoErr, ok := err.(*echo.HTTPError)
			require.True(t, ok)

			assert.Equal(t, http.StatusBadRequest, echoErr.Code)
			gotErr, ok := echoErr.Message.(rosetta.Error)
			require.True(t, ok)

			assert.Equal(t, configuration.ErrorInvalidFormat, gotErr.ErrorDefinition)
		})
	}
}
