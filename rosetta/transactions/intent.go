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
	"fmt"
	"strings"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/object"
)

// Intent describes the intent of an array of Rosetta operations.
type Intent struct {
	From     flow.Address
	To       flow.Address
	Amount   cadence.UFix64
	Payer    flow.Address
	Proposer flow.Address

	ReferenceBlock flow.Identifier
	SequenceNumber uint64
	GasLimit       uint64
}

// DeriveIntent derives a transaction Intent from two operations given as input.
// Specified operations should be symmetrical, a deposit and a withdrawal from two
// different accounts. At the moment, the only fields taken into account are the
// account IDs, amounts and type of operation.
func (p *Parser) DeriveIntent(operations []object.Operation) (*Intent, error) {

	if len(operations) != 2 {
		return nil, failure.InvalidOperations{
			Description: failure.NewDescription("invalid number of operations"),
			Count:       len(operations),
		}
	}

	firstNegative := strings.HasPrefix(operations[0].Amount.Value, "-")
	secondNegative := strings.HasPrefix(operations[1].Amount.Value, "-")

	// Operations are invalid if both operations have the same sign.
	if firstNegative == secondNegative {

		return nil, failure.InvalidIntent{
			Description: failure.NewDescription("invalid operations - values have the same sign"),
			Sender:      operations[0].AccountID.Address,
			Receiver:    operations[1].AccountID.Address,
		}
	}

	// Assume the first operation is the one with the negative amount.
	send := operations[0]
	receive := operations[1]

	// If that was not the case, switch the send and receive operations.
	if !strings.HasPrefix(send.Amount.Value, "-") {
		receive = operations[0]
		send = operations[1]
	}

	// Validate the sender and the receiver account IDs.
	err := p.validate.Account(send.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid sender account: %w", err)
	}
	err = p.validate.Account(receive.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver account: %w", err)
	}

	// Validate the currencies specified for deposit and withdrawal.
	send.Amount.Currency, err = p.validate.Currency(send.Amount.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid sender currency: %w", err)
	}
	receive.Amount.Currency, err = p.validate.Currency(receive.Amount.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver currency: %w", err)
	}

	// Make sure that both the send and receive operations use the same currency.
	// This is perhaps unnecessary at the moment since we only have a single currency.
	if send.Amount.Currency != receive.Amount.Currency {
		return nil, failure.InvalidIntent{
			Sender:      send.AccountID.Address,
			Receiver:    receive.AccountID.Address,
			Description: failure.NewDescription("send and receive currencies do not match"),
		}
	}

	// Parse value specified by the sender, after removing the negative sign prefix.
	trimmed := strings.TrimPrefix(send.Amount.Value, "-")
	sv, err := convert.ParseRosettaValue(trimmed)
	if err != nil {
		return nil, failure.InvalidIntent{
			Sender:   send.AccountID.Address,
			Receiver: receive.AccountID.Address,
			Description: failure.NewDescription("could not parse withdrawal amount",
				failure.WithString("withdrawal_amount", send.Amount.Value),
				failure.WithErr(err),
			),
		}
	}

	// Parse value specified by the receiver.
	rv, err := convert.ParseRosettaValue(receive.Amount.Value)
	if err != nil {
		return nil, failure.InvalidIntent{
			Sender:   send.AccountID.Address,
			Receiver: receive.AccountID.Address,
			Description: failure.NewDescription("could not parse deposit amount",
				failure.WithString("deposit_amount", receive.Amount.Value),
				failure.WithErr(err),
			),
		}
	}

	// Check if the specified amounts match.
	if sv != rv {
		return nil, failure.InvalidIntent{
			Sender:   send.AccountID.Address,
			Receiver: receive.AccountID.Address,
			Description: failure.NewDescription("deposit and withdrawal amounts do not match",
				failure.WithString("deposit_amount", receive.Amount.Value),
				failure.WithString("withdrawal_amount", send.Amount.Value),
			),
		}
	}

	// Validate that the specified operations are transfers.
	if strings.ToUpper(send.Type) != dps.OperationTransfer ||
		strings.ToUpper(receive.Type) != dps.OperationTransfer {

		return nil, failure.InvalidIntent{
			Sender:   send.AccountID.Address,
			Receiver: receive.AccountID.Address,
			Description: failure.NewDescription("only transfer operations are supported",
				failure.WithString("deposit_type", receive.Type),
				failure.WithString("withdrawal_type", send.Type),
			),
		}
	}

	intent := Intent{
		From:     flow.HexToAddress(send.AccountID.Address),
		To:       flow.HexToAddress(receive.AccountID.Address),
		Amount:   sv,
		Payer:    flow.HexToAddress(send.AccountID.Address),
		Proposer: flow.HexToAddress(send.AccountID.Address),
		GasLimit: flow.DefaultMaxTransactionGasLimit,
	}

	return &intent, nil
}
