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

package transactions

import (
	"encoding/hex"
	"fmt"

	"github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/object"
)

// AttachSignature adds the given signature to the transaction.
func (p *Parser) AttachSignature(tx *flow.Transaction, signature object.Signature) (*flow.Transaction, error) {

	// Validate the transaction actors. We expect a single authorizer - the sender account.
	if len(tx.Authorizers) != authorizersRequired {
		return nil, failure.InvalidAuthorizers{
			Have:        uint(len(tx.Authorizers)),
			Want:        authorizersRequired,
			Description: failure.NewDescription("invalid number of authorizers"),
		}
	}

	sender := tx.Authorizers[0]

	// Verify that the sender is the payer, since it is the payer that needs to sign the envelope.
	if tx.Payer != sender {
		return nil, failure.InvalidPayer{
			Have:        convertAddress(tx.Payer),
			Want:        convertAddress(sender),
			Description: failure.NewDescription("invalid transaction payer"),
		}
	}

	// Verify that we do not already have signatures.
	if len(tx.EnvelopeSignatures) > 0 {
		return nil, failure.InvalidSignature{
			Description: failure.NewDescription("unexpected envelope signatures found",
				failure.WithInt("signatures", len(tx.EnvelopeSignatures))),
		}
	}

	// Verify that the signature belongs to the sender.
	signer := flow.HexToAddress(signature.SigningPayload.AccountID.Address)
	if signer != sender {
		return nil, failure.InvalidSignature{
			Description: failure.NewDescription("invalid signer account",
				failure.WithString("have_signer", signer.String()),
				failure.WithString("want_signer", sender.String())),
		}
	}

	bytes, err := hex.DecodeString(signature.HexBytes)
	if err != nil {
		return nil, failure.InvalidSignature{
			Description: failure.NewDescription("invalid signature payload",
				failure.WithErr(err)),
		}
	}

	// Copy the transaction and add the signature.
	signedTx, err := flow.DecodeTransaction(tx.Encode())
	if err != nil {
		return nil, fmt.Errorf("could not copy transaction: %w", err)
	}

	// TODO: allow arbitrary key index
	// => https://github.com/optakt/flow-dps/issues/369
	signedTx.AddEnvelopeSignature(signer, 0, bytes)

	return signedTx, nil
}
