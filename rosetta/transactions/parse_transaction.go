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

// ParseTransactions processes the flow transaction and translates it to a list of operations and a list of
// signers.
func (p *Parser) ParseTransaction(tx *flow.Transaction) ([]object.Operation, []identifier.Account, error) {

	// Verify that we have the correct number of authorizers.
	if len(tx.Authorizers) != authorizersRequired {
		return nil, nil, failure.InvalidAuthorizers{
			Have:        uint(len(tx.Authorizers)),
			Want:        authorizersRequired,
			Description: failure.NewDescription("invalid number of authorizers"),
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

	// Parse the amount script argument.
	val, err := json.Decode(args[0])
	if err != nil {
		return nil, nil, failure.InvalidAmount{
			RawAmount: string(args[0]),
			Description: failure.NewDescription("could not parse transaction amount",
				failure.WithErr(err)),
		}
	}
	amountArg, ok := val.ToGoValue().(uint64)
	if !ok {
		return nil, nil, failure.InvalidAmount{
			RawAmount:   string(args[0]),
			Description: failure.NewDescription("invalid amount"),
		}
	}
	amount := strconv.FormatUint(amountArg, 10)

	// Parse the receiver script argument.
	val, err = json.Decode(args[1])
	if err != nil {
		return nil, nil, failure.InvalidReceiver{
			RawReceiver: string(args[1]),
			Description: failure.NewDescription("could not parse transaction receiver address",
				failure.WithErr(err)),
		}
	}
	receiver := flow.HexToAddress(val.String())

	ops := make([]object.Operation, 2)

	// Create the send operation.
	ops[0] = object.Operation{
		ID: identifier.Operation{
			Index: 0,
		},
		RelatedIDs: []identifier.Operation{{Index: 1}},
		AccountID: identifier.Account{
			Address: tx.Authorizers[0].String(),
		},
		Type: dps.OperationTransfer,
		Amount: object.Amount{
			Value: "-" + amount,
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}

	// Create the receive operation.
	ops[1] = object.Operation{
		ID: identifier.Operation{
			Index: 1,
		},
		RelatedIDs: []identifier.Operation{{Index: 0}},
		AccountID: identifier.Account{
			Address: receiver.String(),
		},
		Type: dps.OperationTransfer,
		Amount: object.Amount{
			Value: amount,
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}

	// Create the signers list.
	signers := make([]identifier.Account, 0)
	for _, sig := range tx.EnvelopeSignatures {
		signer := identifier.Account{
			Address: sig.Address.String(),
		}
		signers = append(signers, signer)
	}

	return ops, signers, nil
}
