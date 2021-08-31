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
	"fmt"
	"sort"
	"strconv"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/object"
)

// DeriveIntent derives a transaction Intent from two operations given as input.
// Specified operations should be symmetrical, a deposit and a withdrawal from two
// different accounts. At the moment, the only fields taken into account are the
// account IDs, amounts and type of operation.
func (t *Transactor) DeriveIntent(operations []object.Operation) (*Intent, error) {

	// Verify that we have exactly two operations.
	if len(operations) != 2 {
		return nil, failure.InvalidOperations{
			Description: failure.NewDescription("invalid number of operations"),
			Count:       len(operations),
		}
	}

	amounts := make(map[int]int64)

	// Parse amounts.
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
	sort.Slice(operations, func(i, j int) bool {
		return amounts[i] < amounts[j]
	})

	send := operations[0]
	receive := operations[1]

	// Validate the currencies specified for deposit and withdrawal.
	sendSymbol, _, err := t.validate.Currency(send.Amount.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid sender currency: %w", err)
	}
	receiveSymbol, _, err := t.validate.Currency(receive.Amount.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver currency: %w", err)
	}

	// Make sure that both the send and receive operations are for FLOW tokens.
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

	amount := amounts[0]
	if amount < 0 {
		amount = -amount
	}

	intent := Intent{
		From:     flow.HexToAddress(send.AccountID.Address),
		To:       flow.HexToAddress(receive.AccountID.Address),
		Amount:   cadence.UFix64(amount),
		Payer:    flow.HexToAddress(send.AccountID.Address),
		Proposer: flow.HexToAddress(send.AccountID.Address),
		GasLimit: flow.DefaultMaxTransactionGasLimit,
	}

	return &intent, nil
}
