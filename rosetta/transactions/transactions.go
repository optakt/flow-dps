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
	"strconv"
	"strings"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/object"
)

// Intent describes the intent of an array of Rosetta operations.
type Intent struct {
	From   flow.Address
	To     flow.Address
	Amount uint64
	Payer  flow.Address // TODO: WIP - for now, we'll treat the sender as the payer to keep things simple
}

// CreateTransfer creates a transaction Intent from two operations given as input.
// Specified operations should be symmetrical, a deposit and a withdrawal from two
// different accounts. At the moment, the only fields taken into account are the
// account IDs, amounts and type of operation.
func (p *Parser) CreateTransfer(operations []object.Operation) (*Intent, error) {

	// TODO: think about naming again - deposit/withdrawal vs sender/receiver
	sender := operations[0]
	receiver := operations[1]

	// The amount is the same, but for the sender the amount will have the '-' prefix.
	// If it doesn't the sender and receiver it means that the operations should be switched.
	if !strings.HasPrefix(sender.Amount.Value, "-") {
		receiver = operations[0]
		sender = operations[1]
	}

	// Validate the sender and the receiver account IDs.
	err := p.validate.Account(sender.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid sender account: %w", err)
	}
	err = p.validate.Account(receiver.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver account: %w", err)
	}

	// Validate the currencies specified for deposit and withdrawal.
	// TODO: check to see if the currencies are actually identical.. At the moment
	// we only support one currency, but in the future we may have multiple.
	sender.Amount.Currency, err = p.validate.Currency(sender.Amount.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid sender currency: %w", err)
	}
	receiver.Amount.Currency, err = p.validate.Currency(receiver.Amount.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver currency: %w", err)
	}

	// Parse value specified by the sender, after removing the negative sign prefix.
	trimmed := strings.TrimPrefix(sender.Amount.Value, "-")
	sv, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return nil, failure.InvalidIntent{
			Sender:   sender.AccountID.Address,
			Receiver: receiver.AccountID.Address,
			Description: failure.NewDescription("could not parse withdrawal amount",
				failure.WithString("withdrawal_amount", sender.Amount.Value),
				failure.WithErr(err),
			),
		}
	}
	// Parse value specified by the receiver.
	rv, err := strconv.ParseUint(receiver.Amount.Value, 10, 64)
	if err != nil {
		return nil, failure.InvalidIntent{
			Sender:   sender.AccountID.Address,
			Receiver: receiver.AccountID.Address,
			Description: failure.NewDescription("could not parse deposit amount",
				failure.WithString("deposit_amount", receiver.Amount.Value),
				failure.WithErr(err),
			),
		}
	}

	// Check if the specified amounts match.
	if sv != rv {
		return nil, failure.InvalidIntent{
			Sender:   sender.AccountID.Address,
			Receiver: receiver.AccountID.Address,
			Description: failure.NewDescription("deposit and withdrawal amounts do not match",
				failure.WithString("deposit_amount", receiver.Amount.Value),
				failure.WithString("withdrawal_amount", sender.Amount.Value),
			),
		}
	}

	if strings.ToUpper(sender.Type) != dps.OperationTransfer ||
		strings.ToUpper(receiver.Type) != dps.OperationTransfer {

		return nil, failure.InvalidIntent{
			Sender:   sender.AccountID.Address,
			Receiver: receiver.AccountID.Address,
			Description: failure.NewDescription("only transfer operations are supported",
				failure.WithString("deposit_type", receiver.Type),
				failure.WithString("withdrawal_type", sender.Type),
			),
		}
	}

	intent := Intent{
		From:   flow.HexToAddress(sender.AccountID.Address),
		To:     flow.HexToAddress(receiver.AccountID.Address),
		Amount: sv,
		Payer:  flow.HexToAddress(sender.AccountID.Address),
	}

	return &intent, nil
}
