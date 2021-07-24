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

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

// ParseTransactions processes the flow transaction and translates it to a list of operations and a list of
// signers.
// TODO: return the list of signers.
func (p *Parser) ParseTransaction(tx flow.Transaction) ([]object.Operation, []identifier.Account, error) {

	ops := make([]object.Operation, 2)

	if len(tx.Authorizers) == 0 {
		return nil, nil, fmt.Errorf("invalid authorizers count: %v", len(tx.Authorizers))
	}

	args := tx.Arguments
	// TODO: convert to Rosetta format
	amount, err := json.Decode(args[0])
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse transaction amount: %w", err)
	}

	// TODO: make sure the amount is correct - should account for 8 decimals
	sendAmount := "-" + amount.String()

	receiver, err := json.Decode(args[1])
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse transaction receiver: %w", err)
	}

	// create the send operation
	ops[0] = object.Operation{
		AccountID: identifier.Account{
			Address: tx.Authorizers[0].Hex(),
		},
		Type: dps.OperationTransfer,
		Amount: object.Amount{
			Value: sendAmount,
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}

	// create the receive operation
	ops[1] = object.Operation{
		AccountID: identifier.Account{
			Address: receiver.String(), // TODO: make sure the format is correct (hex)
		},
		Type: dps.OperationTransfer,
		Amount: object.Amount{
			Value: amount.String(),
			Currency: identifier.Currency{
				Symbol:   dps.FlowSymbol,
				Decimals: dps.FlowDecimals,
			},
		},
	}

	return ops, nil, nil
}
