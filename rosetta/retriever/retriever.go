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
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/models/index"
	"github.com/optakt/flow-dps/rosetta/configuration"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/rosetta"
)

type Retriever struct {
	params    dps.Params
	index     index.Reader
	validate  Validator
	generator Generator
	invoke    Invoker
}

func New(params dps.Params, index index.Reader, validate Validator, generator Generator, invoke Invoker) *Retriever {

	r := Retriever{
		params:    params,
		index:     index,
		validate:  validate,
		generator: generator,
		invoke:    invoke,
	}

	return &r
}

func (r *Retriever) Oldest() (identifier.Block, time.Time, error) {

	first, err := r.index.First()
	if err != nil {
		return identifier.Block{}, time.Time{}, nil
	}

	header, err := r.index.Header(first)
	if err != nil {
		return identifier.Block{}, time.Time{}, nil
	}

	block := identifier.Block{
		Hash:  header.ID().String(),
		Index: header.Height,
	}

	return block, header.Timestamp, nil
}

func (r *Retriever) Current() (identifier.Block, time.Time, error) {

	last, err := r.index.Last()
	if err != nil {
		return identifier.Block{}, time.Time{}, nil
	}

	header, err := r.index.Header(last)
	if err != nil {
		return identifier.Block{}, time.Time{}, nil
	}

	block := identifier.Block{
		Hash:  header.ID().String(),
		Index: header.Height,
	}

	return block, header.Timestamp, nil
}

func (r *Retriever) Balances(block identifier.Block, account identifier.Account, currencies []identifier.Currency) ([]rosetta.Amount, error) {

	// Run validation on the block ID. This also fills in missing information.
	err := r.validate.Block(&block)
	if err != nil {
		return nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Run validation on the account ID. This uses the chain ID to check the
	// address validation.
	err = r.validate.Account(account)
	if err != nil {
		return nil, fmt.Errorf("could not validate account: %w", err)
	}

	// Run validation on the currencies. This checks basically if we know the
	// currency and if it has the correct decimals set, if they are set.
	for _, currency := range currencies {
		err := r.validate.Currency(&currency)
		if err != nil {
			return nil, fmt.Errorf("could not validate currency: %w", err)
		}
	}

	// get the cadence value that is the result of the script execution
	amounts := make([]rosetta.Amount, 0, len(currencies))
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
		amount := rosetta.Amount{
			Currency: currency,
			Value:    strconv.FormatUint(balance, 10),
		}
		amounts = append(amounts, amount)
	}

	return amounts, nil
}

func (r *Retriever) Block(id identifier.Block) (*rosetta.Block, []identifier.Transaction, error) {

	// Run validation on the block ID. This also fills in missing information.
	err := r.validate.Block(&id)
	if err != nil {
		return nil, nil, fmt.Errorf("could not validate block: %w", err)
	}

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
	events, err := r.index.Events(id.Index, flow.EventType(deposit), flow.EventType(withdrawal))
	if err != nil {
		return nil, nil, fmt.Errorf("could not get events: %w", err)
	}

	// Next, we step through all the transactions and accumulate events by transaction ID.
	// NOTE: We consider transactions that don't generate any fund movements as irrelevant for now.
	buckets := make(map[flow.Identifier][]rosetta.Operation)
	for _, event := range events {

		// Decode the event payload into a Cadence value and cast to Cadence event.
		value, err := json.Decode(event.Payload)
		if err != nil {
			return nil, nil, fmt.Errorf("could not decode event: %w", err)
		}
		e, ok := value.(cadence.Event)
		if !ok {
			return nil, nil, fmt.Errorf("could not cast event: %w", err)
		}

		// Check we have the necessary amount of fields.
		if len(e.Fields) != 2 {
			return nil, nil, fmt.Errorf("invalid number of fields (want: %d, have: %d)", 2, len(e.Fields))
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
		op := rosetta.Operation{
			ID: identifier.Operation{
				Index: uint(event.EventIndex),
			},
			RelatedIDs: nil,
			Type:       "TRANSFER",
			Status:     "COMPLETED",
			AccountID: identifier.Account{
				Address: address.String(),
			},
			Amount: rosetta.Amount{
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
	var transactions []*rosetta.Transaction
	for transactionID, operations := range buckets {
		transaction := rosetta.Transaction{
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
	block := rosetta.Block{
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

	// TODO: When a block contains too many transactions, we should limit the
	// size of the returned transaction slice and provide a list of extra
	// transactions IDs in the second return value instead:
	// => https://github.com/optakt/flow-dps/issues/149

	return &block, nil, nil
}

func (r *Retriever) Transaction(block identifier.Block, id identifier.Transaction) (*rosetta.Transaction, error) {

	// TODO: We should start indexing all of the transactions for each block, so
	// that we can actually check transaction existence and return transactions,
	// even if they don't move any funds:
	// => https://github.com/optakt/flow-dps/issues/156

	// Run validation on the block ID. This also fills in missing information.
	err := r.validate.Block(&block)
	if err != nil {
		return nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Retrieve the Flow token default withdrawal and deposit events.
	deposit, err := r.generator.TokensDeposited(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate deposit event type: %w", err)
	}
	withdrawal, err := r.generator.TokensWithdrawn(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate withdrawal event type: %w", err)
	}

	// Retrieve the deposit and withdrawal events for the block (yes, all of them).
	events, err := r.index.Events(block.Index, flow.EventType(deposit), flow.EventType(withdrawal))
	if err != nil {
		return nil, fmt.Errorf("could not get events: %w", err)
	}

	// Go through the events, but only look at the ones with the given transaction ID.
	transaction := rosetta.Transaction{
		ID:         id,
		Operations: []rosetta.Operation{},
	}
	for _, event := range events {

		// Decode the event payload into a Cadence value and cast to Cadence event.
		value, err := json.Decode(event.Payload)
		if err != nil {
			return nil, fmt.Errorf("could not decode event: %w", err)
		}
		e, ok := value.(cadence.Event)
		if !ok {
			return nil, fmt.Errorf("could not cast event: %w", err)
		}

		// Check we have the necessary amount of fields.
		if len(e.Fields) != 2 {
			return nil, fmt.Errorf("invalid number of fields (want: %d, have: %d)", 2, len(e.Fields))
		}

		// Now we have access to the fields for the events; the first one is always
		// the amount, the second one the address. The types coming from Cadence
		// are not native Flow types, so we need to use primitive types first.
		vAmount := e.Fields[0].ToGoValue()
		uAmount, ok := vAmount.(uint64)
		if !ok {
			return nil, fmt.Errorf("could not cast amount (%T)", vAmount)
		}
		vAddress := e.Fields[1].ToGoValue()
		bAddress, ok := vAddress.([flow.AddressLength]byte)
		if !ok {
			return nil, fmt.Errorf("could not cast address (%T)", vAddress)
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
		op := rosetta.Operation{
			ID: identifier.Operation{
				Index: uint(event.EventIndex),
			},
			RelatedIDs: nil,
			Type:       configuration.OperationTransfer,
			Status:     configuration.StatusCompleted.Status,
			AccountID: identifier.Account{
				Address: address.String(),
			},
			Amount: rosetta.Amount{
				Value: strconv.FormatInt(amount, 10),
				Currency: identifier.Currency{
					Symbol:   dps.FlowSymbol,
					Decimals: dps.FlowDecimals,
				},
			},
		}

		// Add the operation to the transaction.
		transaction.Operations = append(transaction.Operations, op)
	}

	// Assign the related operation IDs.
	for _, op := range transaction.Operations {
		for index := range transaction.Operations {
			if transaction.Operations[index].ID != op.ID {
				transaction.Operations[index].RelatedIDs = append(transaction.Operations[index].RelatedIDs, op.ID)
			}
		}
	}

	return &transaction, nil
}
