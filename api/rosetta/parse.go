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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type ParseRequest struct {
	NetworkID   identifier.Network `json:"network_identifier"`
	Signed      bool               `json:"signed"`
	Transaction string             `json:"transaction"`
}

type ParseResponse struct {
	Operations []object.Operation   `json:"operations"`
	SignerIDs  []identifier.Account `json:"account_identifier_signers"`
	Metadata   object.Metadata      `json:"metadata"`
}

func (c *Construction) Parse(ctx echo.Context) error {

	var req ParseRequest
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

	if req.Transaction == "" {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("transaction text empty"))
	}

	var tx flow.Transaction
	err = json.Unmarshal([]byte(req.Transaction), &tx)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("could not decode transaction"))
	}

	operations, signers, err := c.parser.ParseTransaction(tx)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("could not parse transaction: %w", err))
	}

	metadata := object.Metadata{
		ReferenceBlockID: identifier.Block{
			Hash: tx.ReferenceBlockID.Hex(),
		},
		SequenceNumber: tx.ProposalKey.SequenceNumber,
	}

	res := ParseResponse{
		Operations: operations,
		SignerIDs:  signers,
		Metadata:   metadata,
	}

	return ctx.JSON(http.StatusOK, res)
}
