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
	"regexp"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/request"
	"github.com/optakt/flow-dps/rosetta/response"
)

func TestAPI_Options(t *testing.T) {
	// TODO: Repair integration tests
	//       See https://github.com/optakt/flow-dps/issues/333
	t.Skip("integration tests disabled until new snapshot is generated")

	db := setupDB(t)
	api := setupAPI(t, db)

	const wantErrorCount = 23

	// verify version string is in the format of x.y.z
	versionRe := regexp.MustCompile(`\d+\.\d+\.\d+`)

	request := request.Options{
		NetworkID: defaultNetwork(),
	}

	rec, ctx, err := setupRecorder(optionsEndpoint, request)

	err = api.Options(ctx)
	require.NoError(t, err)

	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

	var options response.Options
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &options))

	assert.Regexp(t, versionRe, options.Version.RosettaVersion)
	assert.Regexp(t, versionRe, options.Version.NodeVersion)
	assert.Regexp(t, versionRe, options.Version.MiddlewareVersion)

	assert.True(t, options.Allow.HistoricalBalanceLookup)

	require.Len(t, options.Allow.OperationStatuses, 1)

	status := options.Allow.OperationStatuses[0]
	assert.Equal(t, status.Status, dps.StatusCompleted)
	assert.True(t, status.Successful)

	require.Len(t, options.Allow.OperationTypes, 1)
	assert.Equal(t, options.Allow.OperationTypes[0], dps.OperationTransfer)

	require.Len(t, options.Allow.Errors, wantErrorCount)

	for i := uint(0); i < wantErrorCount; i++ {
		rosettaErr := options.Allow.Errors[i]

		expectedCode := i + 1 // error codes start from 1
		assert.Equal(t, expectedCode, rosettaErr.Code)

		switch expectedCode {
		case configuration.ErrorInternal.Code:
			assert.Equal(t, configuration.ErrorInternal.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInternal.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidEncoding.Code:
			assert.Equal(t, configuration.ErrorInvalidEncoding.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidEncoding.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidFormat.Code:
			assert.Equal(t, configuration.ErrorInvalidFormat.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidFormat.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidNetwork.Code:
			assert.Equal(t, configuration.ErrorInvalidNetwork.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidNetwork.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidAccount.Code:
			assert.Equal(t, configuration.ErrorInvalidAccount.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidAccount.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidCurrency.Code:
			assert.Equal(t, configuration.ErrorInvalidCurrency.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidCurrency.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidBlock.Code:
			assert.Equal(t, configuration.ErrorInvalidBlock.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidBlock.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidTransaction.Code:
			assert.Equal(t, configuration.ErrorInvalidTransaction.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidTransaction.Retriable, rosettaErr.Retriable)

		case configuration.ErrorUnknownBlock.Code:
			assert.Equal(t, configuration.ErrorUnknownBlock.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorUnknownBlock.Retriable, rosettaErr.Retriable)

		case configuration.ErrorUnknownCurrency.Code:
			assert.Equal(t, configuration.ErrorUnknownCurrency.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorUnknownCurrency.Retriable, rosettaErr.Retriable)

		case configuration.ErrorUnknownTransaction.Code:
			assert.Equal(t, configuration.ErrorUnknownTransaction.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorUnknownTransaction.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidIntent.Code:
			assert.Equal(t, configuration.ErrorInvalidIntent.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidIntent.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidAuthorizers.Code:
			assert.Equal(t, configuration.ErrorInvalidAuthorizers.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidAuthorizers.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidPayer.Code:
			assert.Equal(t, configuration.ErrorInvalidPayer.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidPayer.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidProposer.Code:
			assert.Equal(t, configuration.ErrorInvalidProposer.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidProposer.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidScript.Code:
			assert.Equal(t, configuration.ErrorInvalidScript.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidScript.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidArguments.Code:
			assert.Equal(t, configuration.ErrorInvalidArguments.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidArguments.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidAmount.Code:
			assert.Equal(t, configuration.ErrorInvalidAmount.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidAmount.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidReceiver.Code:
			assert.Equal(t, configuration.ErrorInvalidReceiver.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidReceiver.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidSignature.Code:
			assert.Equal(t, configuration.ErrorInvalidSignature.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidSignature.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidKey.Code:
			assert.Equal(t, configuration.ErrorInvalidKey.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidKey.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidPayload.Code:
			assert.Equal(t, configuration.ErrorInvalidPayload.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidPayload.Retriable, rosettaErr.Retriable)

		case configuration.ErrorInvalidSignatures.Code:
			assert.Equal(t, configuration.ErrorInvalidSignatures.Message, rosettaErr.Message)
			assert.Equal(t, configuration.ErrorInvalidSignatures.Retriable, rosettaErr.Retriable)

		default:
			t.Errorf("unknown rosetta error received: (code: %v, message: '%v', retriable: %v", rosettaErr.Code, rosettaErr.Message, rosettaErr.Retriable)
		}
	}
}

func TestAPI_OptionsHandlesErrors(t *testing.T) {
	// TODO: Repair integration tests
	//       See https://github.com/optakt/flow-dps/issues/333
	t.Skip("integration tests disabled until new snapshot is generated")

	db := setupDB(t)
	api := setupAPI(t, db)

	tests := []struct {
		name string

		request request.Options

		checkError assert.ErrorAssertionFunc
	}{
		{
			name: "missing blockchain",
			request: request.Options{
				NetworkID: identifier.Network{
					Blockchain: "",
					Network:    dps.FlowTestnet.String(),
				},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid blockchain",
			request: request.Options{
				NetworkID: identifier.Network{
					Blockchain: invalidBlockchain,
					Network:    dps.FlowTestnet.String(),
				},
			},

			checkError: checkRosettaError(http.StatusUnprocessableEntity, configuration.ErrorInvalidNetwork),
		},
		{
			name: "missing network",
			request: request.Options{
				NetworkID: identifier.Network{
					Blockchain: dps.FlowBlockchain,
					Network:    "",
				},
			},

			checkError: checkRosettaError(http.StatusBadRequest, configuration.ErrorInvalidFormat),
		},
		{
			name: "invalid network",
			request: request.Options{
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

func TestAPI_OptionsHandlesMalformedRequest(t *testing.T) {
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

			err = api.Options(ctx)
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
