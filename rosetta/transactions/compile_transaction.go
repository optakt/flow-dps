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

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/object"
)

// CompileTransaction creates a complete Flow transaction from the given intent and metadata.
func (p *Parser) CompileTransaction(intent *Intent, metadata object.Metadata) (*flow.Transaction, error) {

	// Run validation on the block ID. This also fills in missing information.
	_, blockID, err := p.validate.Block(metadata.ReferenceBlockID)
	if err != nil {
		return nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Generate script for the token transfer.
	script, err := p.generate.TransferTokens(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate transfer script: %w", err)
	}

	// TODO: Allow arbitrary proposal key index
	// => https://github.com/optakt/flow-dps/issues/369

	// Create the transaction.
	tx := flow.NewTransaction().
		SetScript(script).
		SetReferenceBlockID(flow.BytesToID(blockID[:])).
		SetPayer(flow.Address(intent.Payer)).
		SetProposalKey(flow.Address(intent.Proposer), 0, metadata.SequenceNumber).
		AddAuthorizer(flow.Address(intent.From)).
		SetGasLimit(intent.GasLimit)

	receiver := cadence.NewAddress(flow.BytesToAddress(intent.To.Bytes()))

	// Add the script arguments - the amount and the receiver.
	_ = tx.AddArgument(intent.Amount)
	_ = tx.AddArgument(receiver)

	return tx, nil
}
