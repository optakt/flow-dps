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

package transactor

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"

	cjson "github.com/onflow/cadence/encoding/json"
	sdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

// TransactionParser is a wrapper around a pointer to a sdk.Transaction which exposes methods to
// individually parse different elements of the transaction.
type TransactionParser struct {
	tx       *sdk.Transaction
	validate Validator
	generate Generator
	invoke   Invoker
}

// BlockID parses the transaction's BlockID.
func (p *TransactionParser) BlockID() (identifier.Block, error) {
	// Validate the reference block identifier.
	refBlockID := identifier.Block{
		Hash: p.tx.ReferenceBlockID.String(),
	}

	height, blockID, err := p.validate.Block(refBlockID)
	if err != nil {
		return identifier.Block{}, fmt.Errorf("invalid reference block: %w", err)
	}

	return rosettaBlockID(height, blockID), nil
}

// Sequence parses the transaction's sequence number.
func (p *TransactionParser) Sequence() uint64 {
	return p.tx.ProposalKey.SequenceNumber
}

// Signers parses the transaction's signer accounts.
func (p *TransactionParser) Signers() ([]identifier.Account, error) {
	// Since we only support sender as the payer/proposer, we never expect any payload signatures.
	if len(p.tx.PayloadSignatures) > 0 {
		return nil, failure.InvalidSignature{
			Description: failure.NewDescription(payloadSigFound,
				failure.WithInt("signatures", len(p.tx.PayloadSignatures))),
		}
	}

	// We may be parsing an unsigned transaction - if that's the case, we're done.
	if len(p.tx.EnvelopeSignatures) == 0 {
		return nil, nil
	}

	// We don't support multiple signatures.
	if len(p.tx.EnvelopeSignatures) > 1 {
		return nil, failure.InvalidSignature{
			Description: failure.NewDescription(envelopeSigCountInvalid,
				failure.WithInt("signatures", len(p.tx.EnvelopeSignatures))),
		}
	}

	// Validate that it is the sender who signed the transaction.
	signer := p.tx.EnvelopeSignatures[0].Address
	authorizer := p.tx.Authorizers[0]
	if signer != authorizer {
		return nil, failure.InvalidSignature{
			Description: failure.NewDescription(signerInvalid,
				failure.WithString("have_signer", signer.String()),
				failure.WithString("want_signer", authorizer.String()),
				failure.WithString("signature", hex.EncodeToString(p.tx.EnvelopeSignatures[0].Signature))),
		}
	}

	// Check that the signature is valid.
	address := flow.BytesToAddress(signer[:])

	rosBlockID := identifier.Block{Hash: p.tx.ReferenceBlockID.Hex()}
	height, _, err := p.validate.Block(rosBlockID)
	if err != nil {
		return nil, fmt.Errorf("could not validate block: %w", err)
	}

	key, err := p.invoke.Key(height, address, 0)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve key: %w", err)
	}

	// NOTE: signature verification is ported from the DefaultSignatureVerifier
	// => https://github.com/onflow/flow-go/blob/master/fvm/crypto/crypto.go
	hasher, err := crypto.NewHasher(key.HashAlgo)
	if err != nil {
		return nil, fmt.Errorf("could not get new hasher: %w", err)
	}

	message := p.tx.EnvelopeMessage()
	message = append(sdk.TransactionDomainTag[:], message...)

	signature := p.tx.EnvelopeSignatures[0].Signature

	valid, err := key.PublicKey.Verify(signature, message, hasher)
	if err != nil {
		return nil, fmt.Errorf("could not verify transaction signature: %w", err)
	}
	if !valid {
		return nil, failure.InvalidSignature{
			Description: failure.NewDescription(sigInvalid,
				failure.WithString("signature", hex.EncodeToString(signature))),
		}
	}

	sender := identifier.Account{
		Address: authorizer.String(),
	}

	// Validate the sender address.
	_, err = p.validate.Account(sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender account: %w", err)
	}

	// Create the signers list.
	signers := []identifier.Account{
		sender,
	}

	return signers, nil
}

// Operations parses the transaction's operations.
func (p *TransactionParser) Operations() ([]object.Operation, error) {
	// Validate the transaction actors. We expect a single authorizer - the sender account.
	// For now, the sender must also be the proposer and the payer for the transaction.
	if len(p.tx.Authorizers) != requiredAuthorizers {
		return nil, failure.InvalidAuthorizers{
			Have:        uint(len(p.tx.Authorizers)),
			Want:        requiredAuthorizers,
			Description: failure.NewDescription(authorizersInvalid),
		}
	}

	authorizer := p.tx.Authorizers[0]
	sender := identifier.Account{
		Address: authorizer.String(),
	}

	// Validate the sender address.
	_, err := p.validate.Account(sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender account: %w", err)
	}

	// Verify that the sender is the payer and the proposer.
	if p.tx.Payer != authorizer {
		return nil, failure.InvalidPayer{
			Have:        flow.BytesToAddress(p.tx.Payer[:]),
			Want:        flow.BytesToAddress(authorizer[:]),
			Description: failure.NewDescription(payerInvalid),
		}
	}
	if p.tx.ProposalKey.Address != authorizer {
		return nil, failure.InvalidProposer{
			Have:        flow.BytesToAddress(p.tx.ProposalKey.Address[:]),
			Want:        flow.BytesToAddress(authorizer[:]),
			Description: failure.NewDescription(proposerInvalid),
		}
	}

	// Verify the transaction script is the token transfer script.
	script, err := p.generate.TransferTokens(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate transfer script: %w", err)
	}
	if !bytes.Equal(script, p.tx.Script) {
		return nil, failure.InvalidScript{
			Script:      string(p.tx.Script),
			Description: failure.NewDescription(scriptInvalid),
		}
	}

	// Verify that the transaction script has the correct number of arguments.
	args := p.tx.Arguments
	if len(args) != requiredArguments {
		return nil, failure.InvalidArguments{
			Have:        uint(len(args)),
			Want:        requiredArguments,
			Description: failure.NewDescription(scriptArgsInvalid),
		}
	}

	// Parse and validate the amount argument.
	val, err := cjson.Decode(args[0])
	if err != nil {
		return nil, failure.InvalidAmount{
			Amount: string(args[0]),
			Description: failure.NewDescription(amountUnparseable,
				failure.WithErr(err)),
		}
	}
	amountArg, ok := val.ToGoValue().(uint64)
	if !ok {
		return nil, failure.InvalidAmount{
			Amount:      string(args[0]),
			Description: failure.NewDescription(amountInvalid),
		}
	}
	amount := strconv.FormatUint(amountArg, 10)

	// Parse and validate receiver script argument.
	val, err = cjson.Decode(args[1])
	if err != nil {
		return nil, failure.InvalidReceiver{
			Receiver: string(args[1]),
			Description: failure.NewDescription(receiverUnparseable,
				failure.WithErr(err)),
		}
	}
	addr := flow.HexToAddress(val.String())
	receiver := identifier.Account{
		Address: addr.String(),
	}
	_, err = p.validate.Account(receiver)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver account: %w", err)
	}

	// Create the send operation.
	sendOp := object.Operation{
		ID: identifier.Operation{
			Index:        0,
			NetworkIndex: nil, // optional, omitted for now
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
		Status: "", // must NOT be set for non-submitted transactions
	}

	// Create the receive operation.
	receiveOp := object.Operation{
		ID: identifier.Operation{
			Index:        1,
			NetworkIndex: nil, // optional, omitted for now
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
		Status: "", // must NOT be set for non-submitted transactions
	}

	ops := []object.Operation{
		sendOp,
		receiveOp,
	}

	return ops, nil
}
