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
)

// CreateTransaction translates the transaction intent to the Flow Transaction struct.
func (p *Parser) CreateTransaction(intent *Intent) (*flow.Transaction, error) {

	script, err := p.generate.TransferTokens(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate transfer script: %w", err)
	}

	tx := flow.NewTransaction().
		SetScript(script).
		SetReferenceBlockID(flow.BytesToID(intent.ReferenceBlock[:])).
		SetPayer(flow.Address(intent.Payer)).
		SetProposalKey(flow.Address(intent.Proposer), 0, intent.ProposerKeySequenceNumber).
		AddAuthorizer(flow.Address(intent.From)).
		SetGasLimit(intent.GasLimit)

	err = tx.AddArgument(intent.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not add amount argument: %w", err)
	}

	receiver := cadence.NewAddress(flow.BytesToAddress(intent.To.Bytes()))
	err = tx.AddArgument(receiver)
	if err != nil {
		return nil, fmt.Errorf("could not add recipient argument: %w", err)
	}

	return tx, nil
}
