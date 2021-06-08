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
	"github.com/onflow/cadence/encoding/json"
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
	deposit, err := r.generator.TokensDeposited(dps.FlowSymbol)
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate deposit event type: %w", err)
	}
	withdrawal, err := r.generator.TokensWithdrawn(dps.FlowSymbol)
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate withdrawal event type: %w", err)
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
	buckets := make(map[flow.Identifier][]object.Operation)
	for _, event := range events {

		// This switch just skips all events except the ones we are explicitly interested in.
		switch event.Type {
		default:
			continue
		case flow.EventType(deposit):
		case flow.EventType(withdrawal):
		}

		// Decode the event payload into a Cadence value and cast to Cadence event.
		value, err := json.Decode(event.Payload)
		if err != nil {
			return nil, nil, fmt.Errorf("could not decode event: %w", err)
		}
		e, ok := value.(cadence.Event)
		if !ok {
			return nil, nil, fmt.Errorf("could not cast event: %w", err)
		}

		// Now we have access to the fields for the events; the first one is always
		// the amount, the second one the address. The types coming from Cadence
		// are not native Flow types, so we need to use primitive types first.
		vAmount := e.Fields[0].ToGoValue()
		uAmount, ok := vAmount.(uint64)
		if !ok {
			return nil, nil, fmt.Errorf("could not cast amount (%T)", vAmount)
		}
		vAddress := e.Fields[1].ToGoValue()
		bAddress, ok := vAddress.([flow.AddressLength]byte)
		if !ok {
			return nil, nil, fmt.Errorf("could not cast address (%T)", vAddress)
		}

		// Then, we can convert the amount to a signed integer so we can invert
		// it and the address to a native Flow address.
		amount := int64(uAmount)
		address := flow.Address(bAddress)

		// For the witdrawal event, we invert the amount into a negative number.
		if event.Type == flow.EventType(withdrawal) {
			amount = -amount
		}

		// Now we have everything to assemble the respective operation.
		op := object.Operation{
			ID: identifier.Operation{
				Index: uint(event.EventIndex),
			},
			RelatedIDs: nil,
			Type:       "TRANSFER",
			Status:     "COMPLETED",
			AccountID: identifier.Account{
				Address: address.String(),
			},
			Amount: object.Amount{
				Value: strconv.FormatInt(amount, 10),
				Currency: identifier.Currency{
					Symbol:   dps.FlowSymbol,
					Decimals: dps.FlowDecimals,
				},
			},
		}

		// We store all operations for a transaction together in a bucket.
		buckets[event.TransactionID] = append(buckets[event.TransactionID], op)
	}

	// Finally, we batch all of the operations together into the transactions.
	var transactions []*object.Transaction
	for transactionID, operations := range buckets {
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
