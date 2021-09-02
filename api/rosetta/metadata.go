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
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

// MetadataRequest implements the request schema for /construction/metadata.
// `Options` object in this request is generated by a call to `/construction/preprocess`,
// and should be sent unaltered as returned by that endpoint.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#request-3
type MetadataRequest struct {
	NetworkID identifier.Network `json:"network_identifier"`
	Options   object.Options     `json:"options"`
}

// MetadataResponse implements the response schema for /construction/metadata.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-3
type MetadataResponse struct {
	Metadata object.Metadata `json:"metadata"`
}

// Metadata implements the /construction/metadata endpoint of the Rosetta Construction API.
// Metadata endpoint returns information required for constructing the transaction.
// For Flow, that information includes the reference block and sequence number. Reference block
// is the last indexed block, and is used to track transaction expiration. Sequence number is
// the proposer account's public key sequence number. Sequence number is incremented for each
// transaction and is used to prevent replay attacks.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#constructionmetadata
func (c *Construction) Metadata(ctx echo.Context) error {

	var req MetadataRequest
	err := ctx.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, invalidEncoding(invalidJSON, err))
	}

	err = c.validate.Request(req)
	if err != nil {
		return validationError(err)
	}

	err = c.config.Check(req.NetworkID)
	if err != nil {
		return echo.NewHTTPError(apiError(networkCheck, err))
	}

	current, _, err := c.retrieve.Current()
	if err != nil {
		return echo.NewHTTPError(apiError(referenceBlockRetrieval, err))
	}

	// TODO: Allow arbitrary proposal key index
	// => https://github.com/optakt/flow-dps/issues/369
	sequence, err := c.retrieve.Sequence(current, req.Options.AccountID, 0)
	if err != nil {
		return echo.NewHTTPError(apiError(sequenceNumberRetrieval, err))
	}

	// In the `parse` endpoint, we parse a transaction to produce the original metadata (and operations).
	// Since we can only deduce the block hash from the transaction, we will omit the block height from
	// the identifier here, to keep the data identical.
	res := MetadataResponse{
		Metadata: object.Metadata{
			CurrentBlockID: current,
			SequenceNumber: sequence,
		},
	}

	return ctx.JSON(http.StatusOK, res)
}
