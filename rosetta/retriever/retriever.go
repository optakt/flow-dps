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
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/models/convert"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/models/index"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
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
		return identifier.Block{}, time.Time{}, fmt.Errorf("could not find first indexed block: %w", err)
	}

	header, err := r.index.Header(first)
	if err != nil {
		return identifier.Block{}, time.Time{}, fmt.Errorf("could not find block header: %w", err)
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
		return identifier.Block{}, time.Time{}, fmt.Errorf("could not find last indexed block: %w", err)
	}

	header, err := r.index.Header(last)
	if err != nil {
		return identifier.Block{}, time.Time{}, fmt.Errorf("could not find block header: %w", err)
	}

	block := identifier.Block{
		Hash:  header.ID().String(),
		Index: header.Height,
	}

	return block, header.Timestamp, nil
}

func (r *Retriever) Balances(block identifier.Block, account identifier.Account, currencies []identifier.Currency) (identifier.Block, []object.Amount, error) {

	// Run validation on the block identifier. This also fills in missing fields, where possible.
	completed, err := r.validate.Block(block)
	if err != nil {
		return identifier.Block{}, nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Run validation on the account ID. This uses the chain ID to check the
	// address validation.
	err = r.validate.Account(account)
	if err != nil {
		return identifier.Block{}, nil, fmt.Errorf("could not validate account: %w", err)
	}

	// Run validation on the currencies. This checks basically if we know the
	// currency and if it has the correct decimals set, if they are set.
	for idx, currency := range currencies {
		completeCurrency, err := r.validate.Currency(currency)
		if err != nil {
			return identifier.Block{}, nil, fmt.Errorf("could not validate currency: %w", err)
		}
		currencies[idx] = completeCurrency
	}

	// get the cadence value that is the result of the script execution
	amounts := make([]object.Amount, 0, len(currencies))
	address := cadence.NewAddress(flow.HexToAddress(account.Address))
	for _, currency := range currencies {
		getBalance, err := r.generator.GetBalance(currency.Symbol)
		if err != nil {
			return identifier.Block{}, nil, fmt.Errorf("could not generate script: %w", err)
		}
		value, err := r.invoke.Script(block.Index, getBalance, []cadence.Value{address})
		if err != nil {
			return identifier.Block{}, nil, fmt.Errorf("could not invoke script: %w", err)
		}
		balance, ok := value.ToGoValue().(uint64)
		if !ok {
			return identifier.Block{}, nil, fmt.Errorf("could not convert balance (type: %T)", value.ToGoValue())
		}
		amount := object.Amount{
			Currency: currency,
			Value:    strconv.FormatUint(balance, 10),
		}
		amounts = append(amounts, amount)
	}

	return completed, amounts, nil
}

func (r *Retriever) Block(id identifier.Block) (*object.Block, []identifier.Transaction, error) {

	// Run validation on the block ID. This also fills in missing information.
	completed, err := r.validate.Block(id)
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
	header, err := r.index.Header(completed.Index)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get header: %w", err)
	}

	// Next, we get all the events for the block to extract deposit and withdrawal events.
	events, err := r.index.Events(completed.Index, flow.EventType(deposit), flow.EventType(withdrawal))
	if err != nil {
		return nil, nil, fmt.Errorf("could not get events: %w", err)
	}

	// Next, we step through all the transactions and accumulate events by transaction ID.
	// NOTE: We consider transactions that don't generate any fund movements as irrelevant for now.
	tmap, err := convert.EventsToTransactions(events, withdrawal)
	if err != nil {
		return nil, nil, err
	}

	var transactions []*object.Transaction
	for _, transaction := range tmap {
		transactions = append(transactions, transaction)
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

	// TODO: When a block contains too many transactions, we should limit the
	// size of the returned transaction slice and provide a list of extra
	// transactions IDs in the second return value instead:
	// => https://github.com/optakt/flow-dps/issues/149

	return &block, nil, nil
}

func (r *Retriever) Transaction(block identifier.Block, id identifier.Transaction) (*object.Transaction, error) {

	// TODO: We should start indexing all of the transactions for each block, so
	// that we can actually check transaction existence and return transactions,
	// even if they don't move any funds:
	// => https://github.com/optakt/flow-dps/issues/156

	// Run validation on the block ID. This also fills in missing information.
	completed, err := r.validate.Block(block)
	if err != nil {
		return nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Run validation on the transaction ID. This should never fail, as we
	// already check the length, but let's run it anyway.
	err = r.validate.Transaction(id)
	if err != nil {
		return nil, fmt.Errorf("could not validate transaction: %w", err)
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
	events, err := r.index.Events(completed.Index, flow.EventType(deposit), flow.EventType(withdrawal))
	if err != nil {
		return nil, fmt.Errorf("could not get events: %w", err)
	}

	transactions, err := convert.EventsToTransactions(events, withdrawal)
	if err != nil {
		return nil, err
	}

	transaction, found := transactions[id.Hash]
	if !found {
		return nil, fmt.Errorf("no transaction found with id %q at block %s", id, block.Hash)
	}

	return transaction, nil
}
