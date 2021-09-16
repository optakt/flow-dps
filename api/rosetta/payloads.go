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
	"github.com/optakt/flow-dps/rosetta/request"
	"github.com/optakt/flow-dps/rosetta/response"
)

// Payloads implements the /construction/payloads endpoint of the Rosetta Construction API.
// It receives an array of operations and all other relevant information required to construct
// an unsigned transaction. Operations must deterministically describe the intent of the
// transaction. Besides the unsigned transaction text, this endpoint also returns the list
// of payloads that should be signed.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#constructionpayloads
func (c *Construction) Payloads(ctx echo.Context) error {

	var req request.Payloads
	err := ctx.Bind(&req)
	if err != nil {
		return unpackError(err)
	}

	err = c.validate.Request(req)
	if err != nil {
		return formatError(err)
	}

	// Metadata object is the response from our metadata endpoint. Thus, the object
	// should be okay, but let's validate it anyway.
	err = c.validate.CompleteBlockID(req.Metadata.CurrentBlockID)
	if err != nil {
		return formatError(err)
	}

	intent, err := c.transact.DeriveIntent(req.Operations)
	if err != nil {
		return apiError(intentDetermination, err)
	}

	unsigned, err := c.transact.CompileTransaction(req.Metadata.CurrentBlockID, intent, req.Metadata.SequenceNumber)
	if err != nil {
		return apiError(txConstruction, err)
	}

	sender := identifier.Account{
		Address: intent.From.String(),
	}
	algo, hash, err := c.transact.HashPayload(req.Metadata.CurrentBlockID, unsigned, sender)
	if err != nil {
		return apiError(payloadHashing, err)
	}

	// We only support a single signer at the moment, so the account only needs to sign the transaction envelope.
	res := response.Payloads{
		Transaction: unsigned,
		Payloads: []object.SigningPayload{
			{
				AccountID:     identifier.Account{Address: intent.From.Hex()},
				HexBytes:      hash,
				SignatureType: algo,
			},
		},
	}

	return ctx.JSON(http.StatusOK, res)
}
