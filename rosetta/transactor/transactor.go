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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/onflow/cadence"
	sdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

const (
	requiredAuthorizers = 1       // we only support one authorizer per transaction
	requiredArguments   = 2       // transactions need to arguments (amount & receiver)
	requiredOperations  = 2       // transactions are made of two operations (deposit & withdrawal)
	requiredAlgorithm   = "ecdsa" // transactions are signed with ECSDA
)

// Transactor can determine the transaction intent from an array of Rosetta
// operations, create a Flow transaction from a transaction intent and
// translate a Flow transaction back to an array of Rosetta operations.
type Transactor struct {
	validate Validator
	generate Generator
	invoke   Invoker
	submit   Submitter
}

// Parser represents something that can parse a transaction into individual parts.
type Parser interface {
	BlockID() (identifier.Block, error)
	Sequence() uint64
	Signers() ([]identifier.Account, error)
	Operations() ([]object.Operation, error)
}

// New creates a new transactor to handle interactions with Flow transactions.
func New(validate Validator, generate Generator, invoke Invoker, submit Submitter) *Transactor {

	p := Transactor{
		validate: validate,
		generate: generate,
		invoke:   invoke,
		submit:   submit,
	}

	return &p
}

// DeriveIntent derives a transaction Intent from two operations given as input.
// Specified operations should be symmetrical, a deposit and a withdrawal from two
// different accounts. At the moment, the only fields taken into account are the
// account IDs, amounts and type of operation.
func (t *Transactor) DeriveIntent(operations []object.Operation) (*Intent, error) {

	// Verify that we have exactly two operations.
	if len(operations) != requiredOperations {
		return nil, failure.InvalidOperations{
			Description: failure.NewDescription("invalid number of operations"),
			Want:        requiredOperations,
			Have:        uint(len(operations)),
		}
	}

	// Parse amounts.
	amounts := make([]int64, requiredOperations)
	for i, op := range operations {
		amount, err := strconv.ParseInt(op.Amount.Value, 10, 64)
		if err != nil {
			return nil, failure.InvalidIntent{
				Description: failure.NewDescription("could not parse amount",
					failure.WithString("amount", op.Amount.Value),
					failure.WithErr(err),
				),
			}
		}
		amounts[i] = amount
	}

	// Verify that the amounts match.
	if amounts[0] != -amounts[1] {
		return nil, failure.InvalidIntent{
			Description: failure.NewDescription("transfer amounts do not match",
				failure.WithString("first_amount", operations[0].Amount.Value),
				failure.WithString("second_amount", operations[1].Amount.Value),
			),
		}
	}

	// Sort the operations so that the send operation (negative amount) comes first.
	sort.Slice(operations, func(i int, j int) bool {
		return amounts[i] < amounts[j]
	})
	sort.Slice(amounts, func(i int, j int) bool {
		return amounts[i] < amounts[j]
	})

	// Validate the currencies specified for deposit and withdrawal.
	send := operations[0]
	receive := operations[1]
	sendSymbol, _, err := t.validate.Currency(send.Amount.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid sender currency: %w", err)
	}
	receiveSymbol, _, err := t.validate.Currency(receive.Amount.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver currency: %w", err)
	}

	// Make sure that both operations are for FLOW tokens.
	if sendSymbol != dps.FlowSymbol || receiveSymbol != dps.FlowSymbol {
		return nil, failure.InvalidIntent{
			Description: failure.NewDescription("invalid currencies found",
				failure.WithString("sender", send.AccountID.Address),
				failure.WithString("receiver", receive.AccountID.Address),
				failure.WithString("withdrawal_currency", send.Amount.Currency.Symbol),
				failure.WithString("deposit_currency", receive.Amount.Currency.Symbol)),
		}
	}

	// Validate the sender and the receiver account IDs.
	_, err = t.validate.Account(send.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid sender account: %w", err)
	}
	_, err = t.validate.Account(receive.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver account: %w", err)
	}

	// Validate that the specified operations are transfers.
	if send.Type != dps.OperationTransfer || receive.Type != dps.OperationTransfer {
		return nil, failure.InvalidIntent{
			Description: failure.NewDescription("only transfer operations are supported",
				failure.WithString("withdrawal_type", send.Type),
				failure.WithString("deposit_type", receive.Type),
			),
		}
	}

	// The smaller amount is first, so the second one should always have the
	// positive number.
	amount := amounts[1]
	intent := Intent{
		From:     flow.HexToAddress(send.AccountID.Address),
		To:       flow.HexToAddress(receive.AccountID.Address),
		Amount:   cadence.UFix64(amount),
		Payer:    flow.HexToAddress(send.AccountID.Address),
		Proposer: flow.HexToAddress(send.AccountID.Address),
	}

	return &intent, nil
}

// CompileTransaction creates a complete Flow transaction from the given intent and metadata.
func (t *Transactor) CompileTransaction(rosBlockID identifier.Block, intent *Intent, sequence uint64) (string, error) {

	// Generate script for the token transfer.
	script, err := t.generate.TransferTokens(dps.FlowSymbol)
	if err != nil {
		return "", fmt.Errorf("could not generate transfer script: %w", err)
	}

	// Create the transaction.
	unsignedTx := sdk.NewTransaction().
		SetScript(script).
		SetReferenceBlockID(sdk.HexToID(rosBlockID.Hash)).
		SetPayer(sdk.Address(intent.Payer)).
		SetProposalKey(sdk.Address(intent.Proposer), 0, sequence).
		AddAuthorizer(sdk.Address(intent.From)).
		SetGasLimit(flow.DefaultMaxTransactionGasLimit)

	receiver := cadence.NewAddress(flow.BytesToAddress(intent.To.Bytes()))

	// Add the script arguments - the amount and the receiver.
	// NOTE: This can only fail if the argument can not be encoded using the
	// Cadence JSON encoder, which will never happen here.
	_ = unsignedTx.AddArgument(intent.Amount)
	_ = unsignedTx.AddArgument(receiver)

	payload, err := t.encodeTransaction(unsignedTx)
	if err != nil {
		return "", fmt.Errorf("could not encode transaction: %w", err)
	}

	return payload, nil
}

// HashPayload returns the algorithm and hash of a given unsigned transaction when signed by
// a given account's public key.
func (t *Transactor) HashPayload(rosBlockID identifier.Block, unsigned string, signer identifier.Account) (string, string, error) {

	unsignedTx, err := t.decodeTransaction(unsigned)
	if err != nil {
		return "", "", fmt.Errorf("could not decode transaction: %w", err)
	}

	// Validate block.
	height, _, err := t.validate.Block(rosBlockID)
	if err != nil {
		return "", "", fmt.Errorf("could not validate block: %w", err)
	}

	// Validate address.
	address, err := t.validate.Account(signer)
	if err != nil {
		return "", "", fmt.Errorf("could not validate account: %w", err)
	}

	key, err := t.invoke.Key(height, address, 0)
	if err != nil {
		return "", "", failure.InvalidKey{
			Description: failure.NewDescription("invalid account key", failure.WithErr(err)),
			Height:      height,
			Address:     address,
			Index:       0,
		}
	}

	message := unsignedTx.EnvelopeMessage()
	message = append(flow.TransactionDomainTag[:], message...)

	hasher, err := crypto.NewHasher(key.HashAlgo)
	if err != nil {
		return "", "", fmt.Errorf("could not create hasher: %w", err)
	}

	hash := hex.EncodeToString(hasher.ComputeHash(message))

	return requiredAlgorithm, hash, nil
}

// AttachSignatures returns the given transaction with the given signatures attached to it.
func (t *Transactor) AttachSignatures(unsigned string, signatures []object.Signature) (string, error) {

	unsignedTx, err := t.decodeTransaction(unsigned)
	if err != nil {
		return "", fmt.Errorf("could not decode transaction: %w", err)
	}

	// Validate the transaction actors. We expect a single authorizer - the sender account.
	if len(unsignedTx.Authorizers) != requiredAuthorizers {
		return "", failure.InvalidAuthorizers{
			Have:        uint(len(unsignedTx.Authorizers)),
			Want:        requiredAuthorizers,
			Description: failure.NewDescription("invalid number of authorizers"),
		}
	}

	// We expect one signature for the one signer.
	if len(unsignedTx.Authorizers) != len(signatures) {
		return "", failure.InvalidSignatures{
			Have:        uint(len(signatures)),
			Want:        uint(len(unsignedTx.Authorizers)),
			Description: failure.NewDescription("invalid number of signatures"),
		}
	}

	// Verify that the sender is the payer, since it is the payer that needs to sign the envelope.
	sender := unsignedTx.Authorizers[0]
	signature := signatures[0]
	if unsignedTx.Payer != sender {
		return "", failure.InvalidPayer{
			Have:        flow.BytesToAddress(unsignedTx.Payer[:]),
			Want:        flow.BytesToAddress(sender[:]),
			Description: failure.NewDescription("invalid transaction payer"),
		}
	}

	// Verify that we do not already have signatures.
	if len(unsignedTx.EnvelopeSignatures) > 0 {
		return "", failure.InvalidSignature{
			Description: failure.NewDescription("unexpected envelope signatures found",
				failure.WithInt("signatures", len(unsignedTx.EnvelopeSignatures))),
		}
	}

	// Verify that the signature belongs to the sender.
	signer := sdk.HexToAddress(signature.SigningPayload.AccountID.Address)
	if signer != sender {
		return "", failure.InvalidSignature{
			Description: failure.NewDescription("invalid signer account",
				failure.WithString("have_signer", signer.Hex()),
				failure.WithString("want_signer", sender.Hex()),
			),
		}
	}

	if signature.SignatureType != requiredAlgorithm {
		return "", failure.InvalidSignature{
			Description: failure.NewDescription("invalid signature algorithm",
				failure.WithString("have_algo", signature.SignatureType),
				failure.WithString("want_algo", requiredAlgorithm),
			),
		}
	}

	bytes, err := hex.DecodeString(signature.HexBytes)
	if err != nil {
		return "", failure.InvalidSignature{
			Description: failure.NewDescription("invalid signature payload",
				failure.WithErr(err)),
		}
	}

	signedTx := unsignedTx.AddEnvelopeSignature(signer, 0, bytes)
	signed, err := t.encodeTransaction(signedTx)
	if err != nil {
		return "", fmt.Errorf("could not encode transaction: %w", err)
	}

	return signed, nil
}

// TransactionIdentifier returns the transaction identifier of a given signed transaction.
func (t *Transactor) TransactionIdentifier(signed string) (identifier.Transaction, error) {

	signedTx, err := t.decodeTransaction(signed)
	if err != nil {
		return identifier.Transaction{}, fmt.Errorf("could not decode transaction: %w", err)
	}

	rosTxID := identifier.Transaction{
		Hash: signedTx.ID().Hex(),
	}

	return rosTxID, nil
}

// SubmitTransaction submits the given signed transaction.
func (t *Transactor) SubmitTransaction(signed string) (identifier.Transaction, error) {

	signedTx, err := t.decodeTransaction(signed)
	if err != nil {
		return identifier.Transaction{}, fmt.Errorf("could not decode transaction: %w", err)
	}

	err = t.submit.Transaction(signedTx)
	if err != nil {
		return identifier.Transaction{}, fmt.Errorf("could not submit transaction: %w", err)
	}

	return rosettaTxID(signedTx.ID()), nil
}

func (t *Transactor) encodeTransaction(tx *sdk.Transaction) (string, error) {

	data, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("could not marshal transaction: %w", err)
	}
	payload := base64.StdEncoding.EncodeToString(data)

	return payload, nil
}

func (t *Transactor) decodeTransaction(payload string) (*sdk.Transaction, error) {

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, failure.InvalidPayload{
			Description: failure.NewDescription(err.Error()),
			Encoding:    "base64",
		}
	}

	var tx sdk.Transaction
	err = json.Unmarshal(data, &tx)
	if err != nil {
		return nil, failure.InvalidPayload{
			Description: failure.NewDescription(err.Error()),
			Encoding:    "json",
		}
	}

	return &tx, nil
}

// Parse processes the flow transaction, validates its correctness and translates it
// to a list of operations and a list of signers.
func (t *Transactor) Parse(payload string) (Parser, error) {
	tx, err := t.decodeTransaction(payload)
	if err != nil {
		return nil, err
	}

	p := TransactionParser{
		tx:       tx,
		validate: t.validate,
		generate: t.generate,
		invoke:   t.invoke,
	}

	return &p, nil
}
