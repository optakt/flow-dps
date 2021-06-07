// Copyright 2021 Alvalor S.A.
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

package retriever

import (
	"fmt"
	"strconv"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/models/index"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type Retriever struct {
	index     index.Reader
	generator Generator
	invoke    Invoker
}

func New(index index.Reader, generator Generator, invoke Invoker) *Retriever {

	r := Retriever{
		index:     index,
		generator: generator,
		invoke:    invoke,
	}

	return &r
}

func (r *Retriever) Balances(network identifier.Network, block identifier.Block, account identifier.Account, currencies []identifier.Currency) ([]object.Amount, error) {

	// get the cadence value that is the result of the script execution
	amounts := make([]object.Amount, 0, len(currencies))
	address := cadence.NewAddress(flow.HexToAddress(account.Address))
	for _, currency := range currencies {
		getBalance, err := r.generator.GetBalance(currency.Symbol)
		if err != nil {
			return nil, fmt.Errorf("could not generate script: %w", err)
		}
		value, err := r.invoke.Script(block.Index, getBalance, []cadence.Value{address})
		if err != nil {
			return nil, fmt.Errorf("could not invoke script: %w", err)
		}
		balance, ok := value.ToGoValue().(uint64)
		if !ok {
			return nil, fmt.Errorf("could not convert balance (type: %T)", value.ToGoValue())
		}
		amount := object.Amount{
			Currency: currency,
			Value:    strconv.FormatUint(balance, 10),
		}
		amounts = append(amounts, amount)
	}

	return amounts, nil
}

func (r *Retriever) Block(network identifier.Network, id identifier.Block) (*object.Block, []identifier.Transaction, error) {

	// Retrieve the Flow token default withdrawal and deposit events.
	withdrawal, err := r.generator.Withdrawal(dps.FlowSymbol)
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate withdrawal event type: %w", err)
	}
	deposit, err := r.generator.Deposit(dps.FlowSymbol)
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate deposit event type: %w", err)
	}

	// Then, we get the header; it will give us the block ID, parent ID and timestamp.
	header, err := r.index.Header(id.Index)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get header: %w", err)
	}

	// Next, we get all the events for the block to extract deposit and withdrawal events.
	events, err := r.index.Events(id.Index)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get events: %w", err)
	}

	// Next, we step through all the transactions and accumulate events by transaction ID.
	// NOTE: We consider transactions that don't generate any fund movements as irrelevant for now.
	batches := make(map[flow.Identifier][]object.Operation)
	for _, event := range events {
		if event.Type != flow.EventType(withdrawal) && event.Type != flow.EventType(deposit) {
			continue
		}
		op := object.Operation{
			ID: identifier.Operation{
				Index: uint(event.EventIndex),
			},
			RelatedIDs: nil, // needs to be set when building transactions
			Type:       "transfer",
			Status:     "sealed",
			AccountID: identifier.Account{
				Address: "", // needs to be set from decoded event
			},
			Amount: object.Amount{
				Value: "", // needs to be set from decoded event
				Currency: identifier.Currency{
					Symbol:   dps.FlowSymbol,
					Decimals: dps.FlowDecimals,
				},
			},
		}
		switch event.Type {
		case flow.EventType(withdrawal):
			// FIXME: decode withdrawal event and get account address & amount
		case flow.EventType(deposit):
			// FIXME: decode deposit event and get account address & amount
		}
		batches[event.TransactionID] = append(batches[event.TransactionID], op)
	}

	// Finally, we batch all of the operations together into the transactions.
	var transactions []*object.Transaction
	for transactionID, operations := range batches {
		transaction := object.Transaction{
			ID: identifier.Transaction{
				Hash: transactionID.String(),
			},
			Operations: operations,
		}
		for _, op := range operations {
			for index := range transaction.Operations {
				if transaction.Operations[index].ID != op.ID {
					transaction.Operations[index].RelatedIDs = append(transaction.Operations[index].RelatedIDs, op.ID)
				}
			}
		}
		transactions = append(transactions, &transaction)
	}

	// Now we just need to build the block.
	block := object.Block{
		ID: identifier.Block{
			Index: header.Height,
			Hash:  header.ID().String(),
		},
		ParentID: identifier.Block{
			Index: header.Height - 1,
			Hash:  header.ParentID.String(),
		},
		Timestamp:    header.Timestamp.UnixNano() / 1_000_000,
		Transactions: transactions,
	}

	// TODO: When a block contains to many transactions / operations, we should
	// limit the returned block size and return a list of transaction IDs in
	// the second field.
	// => https://github.com/optakt/flow-dps/issues/149

	return &block, nil, nil
}

func (r *Retriever) Transaction(network identifier.Network, block identifier.Block, transaction identifier.Transaction) (*object.Transaction, error) {
	// TODO: implement Rosetta transaction retrieval
	// => https://github.com/optakt/flow-dps/issues/44
	return nil, fmt.Errorf("not implemented")
}
