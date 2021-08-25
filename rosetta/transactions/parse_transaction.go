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
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

const (
	// Transactions should have exactly one authorizer.
	authorizersRequired = 1

	// Transaction script should have exactly two arguments.
	argsRequired = 2
)

// ParseTransactions processes the flow transaction, validates its correctness and translates it
// to a list of operations and a list of signers.
func (p *Parser) ParseTransaction(tx *flow.Transaction) ([]object.Operation, []identifier.Account, error) {

	// Validate the transaction actors. We expect a single authorizer - the sender account.
	// For now, the sender must also be the proposer and the payer for the transaction.

	if len(tx.Authorizers) != authorizersRequired {
		return nil, nil, failure.InvalidAuthorizers{
			Have:        uint(len(tx.Authorizers)),
			Want:        authorizersRequired,
			Description: failure.NewDescription("invalid number of authorizers"),
		}
	}

	authorizer := tx.Authorizers[0]
	sender := identifier.Account{
		Address: authorizer.String(),
	}

	// Validate the sender address.
	_, err := p.validate.Account(sender)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid sender account: %w", err)
	}

	// Verify that the sender is the payer and the proposer.
	if tx.Payer != authorizer {
		return nil, nil, failure.InvalidPayer{
			Have:        convertAddress(tx.Payer),
			Want:        convertAddress(authorizer),
			Description: failure.NewDescription("invalid transaction payer"),
		}
	}
	if tx.ProposalKey.Address != authorizer {
		return nil, nil, failure.InvalidProposer{
			Have:        convertAddress(tx.ProposalKey.Address),
			Want:        convertAddress(authorizer),
			Description: failure.NewDescription("invalid transaction proposer"),
		}
	}

	// Verify the transaction script is the token transfer script.
	script, err := p.generate.TransferTokens(dps.FlowSymbol)
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate transfer script: %w", err)
	}
	if !bytes.Equal(script, tx.Script) {
		return nil, nil, failure.InvalidScript{
			Script:      string(tx.Script),
			Description: failure.NewDescription("transaction text is not valid token transfer script"),
		}
	}

	// Verify that the transaction script has the correct number of arguments.
	args := tx.Arguments
	if len(args) != argsRequired {
		return nil, nil, failure.InvalidArguments{
			Have:        uint(len(args)),
			Want:        argsRequired,
			Description: failure.NewDescription("invalid number of arguments"),
		}
	}

	// Parse and validate the amount argument.
	val, err := json.Decode(args[0])
	if err != nil {
		return nil, nil, failure.InvalidAmount{
			Amount: string(args[0]),
			Description: failure.NewDescription("could not parse transaction amount",
				failure.WithErr(err)),
		}
	}
	amountArg, ok := val.ToGoValue().(uint64)
	if !ok {
		return nil, nil, failure.InvalidAmount{
			Amount:      string(args[0]),
			Description: failure.NewDescription("invalid amount"),
		}
	}
	amount := strconv.FormatUint(amountArg, 10)

	// Parse and validate receiver script argument.
	val, err = json.Decode(args[1])
	if err != nil {
		return nil, nil, failure.InvalidReceiver{
			Receiver: string(args[1]),
			Description: failure.NewDescription("could not parse transaction receiver address",
				failure.WithErr(err)),
		}
	}
	receiver := identifier.Account{
		Address: val.String(),
	}
	_, err = p.validate.Account(receiver)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid receiver account: %w", err)
	}

	// Validate the reference block identifier.
	rosBlockID := identifier.Block{
		Hash: tx.ReferenceBlockID.String(),
	}
	_, _, err = p.validate.Block(rosBlockID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid reference block: %w", err)
	}

	// Create the send operation.
	sendOp := object.Operation{
		ID: identifier.Operation{
			Index: 0,
		},
		AccountID: sender,
		Type:      dps.OperationTransfer,
		Amount: object.Amount{
			Value: "-" + amount,
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}

	// Create the receive operation.
	receiveOp := object.Operation{
		ID: identifier.Operation{
			Index: 1,
		},
		AccountID: receiver,
		Type:      dps.OperationTransfer,
		Amount: object.Amount{
			Value: amount,
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}

	// Create the operations list.
	ops := []object.Operation{
		sendOp,
		receiveOp,
	}

	// Since we only support sender as the payer/proposer, we never expect any payload signatures.
	if len(tx.PayloadSignatures) > 0 {
		return nil, nil, failure.InvalidSignature{
			Description: failure.NewDescription("unexpected payload signature found",
				failure.WithInt("signatures", len(tx.PayloadSignatures))),
		}
	}

	// We may be parsing an unsigned transaction - if that's the case, we're done.
	if len(tx.EnvelopeSignatures) == 0 {
		return ops, nil, nil
	}

	// We don't support multiple signatures.
	if len(tx.EnvelopeSignatures) > 1 {
		return nil, nil, failure.InvalidSignature{
			Description: failure.NewDescription("unexpected envelope signatures found",
				failure.WithInt("signatures", len(tx.EnvelopeSignatures))),
		}
	}

	// Validate that it is the sender who signed the transaction.
	signer := tx.EnvelopeSignatures[0].Address
	if signer != authorizer {
		return nil, nil, failure.InvalidSignature{
			Description: failure.NewDescription("invalid signer account",
				failure.WithString("have_signer", signer.String()),
				failure.WithString("want_signer", authorizer.String()),
				failure.WithString("signature", hex.EncodeToString(tx.EnvelopeSignatures[0].Signature))),
		}
	}

	// Create the signers list.
	signers := []identifier.Account{
		sender,
	}

	return ops, signers, nil
}
