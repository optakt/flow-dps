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

package retriever

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/rosetta/converter"
	"github.com/optakt/flow-dps/rosetta/failure"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
)

type Retriever struct {
	cfg Config

	params    dps.Params
	index     dps.Reader
	validate  Validator
	generator Generator
	invoke    Invoker
	convert   Converter
}

func New(params dps.Params, index dps.Reader, validate Validator, generator Generator, invoke Invoker, convert Converter, options ...func(*Config)) *Retriever {

	cfg := Config{
		TransactionLimit: 200,
	}

	for _, opt := range options {
		opt(&cfg)
	}

	r := Retriever{
		cfg:       cfg,
		params:    params,
		index:     index,
		validate:  validate,
		generator: generator,
		invoke:    invoke,
		convert:   convert,
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
		Index: &header.Height,
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
		Index: &header.Height,
	}

	return block, header.Timestamp, nil
}

func (r *Retriever) Balances(blockQualifier identifier.Block, accountQualifier identifier.Account, currencyQualifiers []identifier.Currency) (identifier.Block, []object.Amount, error) {

	// Run validation on the block qualifier. This also fills in missing fields, where possible.
	completed, err := r.validate.Block(blockQualifier)
	if err != nil {
		return identifier.Block{}, nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Run validation on the account qualifier. This uses the chain ID to check the
	// address validation.
	err = r.validate.Account(accountQualifier)
	if err != nil {
		return identifier.Block{}, nil, fmt.Errorf("could not validate account: %w", err)
	}

	// Run validation on the currency qualifiers. This checks basically if we know the
	// currency and if it has the correct decimals set, if they are set.
	for idx, currency := range currencyQualifiers {
		completeCurrency, err := r.validate.Currency(currency)
		if err != nil {
			return identifier.Block{}, nil, fmt.Errorf("could not validate currency: %w", err)
		}
		currencyQualifiers[idx] = completeCurrency
	}

	// Get the Cadence value that is the result of the script execution.
	amounts := make([]object.Amount, 0, len(currencyQualifiers))
	address := cadence.NewAddress(flow.HexToAddress(accountQualifier.Address))
	for _, currency := range currencyQualifiers {
		getBalance, err := r.generator.GetBalance(currency.Symbol)
		if err != nil {
			return identifier.Block{}, nil, fmt.Errorf("could not generate script: %w", err)
		}
		value, err := r.invoke.Script(*completed.Index, getBalance, []cadence.Value{address})
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

func (r *Retriever) Block(blockQualifier identifier.Block) (*object.Block, []identifier.Transaction, error) {

	// Run validation on the block ID. This also fills in missing information.
	completed, err := r.validate.Block(blockQualifier)
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

	// Then, get the header; it contains the block ID, parent ID and timestamp.
	header, err := r.index.Header(*completed.Index)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get header: %w", err)
	}

	// Next, we get all the events for the block to extract deposit and withdrawal events.
	events, err := r.index.Events(*completed.Index, flow.EventType(deposit), flow.EventType(withdrawal))
	if err != nil {
		return nil, nil, fmt.Errorf("could not get events: %w", err)
	}

	// Convert events to operations and group them by transaction ID.
	buckets := make(map[string][]object.Operation)
	for _, event := range events {
		op, err := r.convert.EventToOperation(event)
		if errors.Is(err, converter.ErrIrrelevant) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("could not convert event: %w", err)
		}

		tID := event.TransactionID.String()
		buckets[tID] = append(buckets[tID], *op)
	}

	// Iterate over all transactionIDs to create transactions with all relevant operations.
	var blockTransactions []*object.Transaction
	var extraTransactions []identifier.Transaction
	var count int
	for txID, operations := range buckets {
		if count >= int(r.cfg.TransactionLimit) {
			extraTransactions = append(extraTransactions, identifier.Transaction{Hash: txID})
			continue
		}

		// Set RelatedIDs for all operations for the same transaction.
		for i := range operations {
			for j := range operations {
				if i == j {
					continue
				}

				operations[i].RelatedIDs = append(operations[i].RelatedIDs, operations[j].ID)
			}
		}

		transaction := object.Transaction{
			ID:         identifier.Transaction{Hash: txID},
			Operations: operations,
		}
		blockTransactions = append(blockTransactions, &transaction)

		count++
	}

	parent := identifier.Block{
		Index: &header.Height,
		Hash:  header.ID().String(),
	}

	// Rosetta spec notes that for genesis block, it is recommended to use the
	// genesis block identifier also for the parent block identifier.
	// See https://www.rosetta-api.org/docs/common_mistakes.html#malformed-genesis-block
	if header.Height > 0 {
		h := header.Height - 1
		parent = identifier.Block{
			Index: &h,
			Hash:  header.ParentID.String(),
		}
	}

	// Now we just need to build the block.
	block := object.Block{
		ID: identifier.Block{
			Index: &header.Height,
			Hash:  header.ID().String(),
		},
		ParentID:     parent,
		Timestamp:    header.Timestamp.UnixNano() / 1_000_000,
		Transactions: blockTransactions,
	}

	return &block, extraTransactions, nil
}

func (r *Retriever) Transaction(blockQualifier identifier.Block, txQualifier identifier.Transaction) (*object.Transaction, error) {

	// Run validation on the block qualifier. This also fills in missing information.
	completed, err := r.validate.Block(blockQualifier)
	if err != nil {
		return nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Run validation on the transaction qualifier. This should never fail, as we
	// already check the length, but let's run it anyway.
	err = r.validate.Transaction(txQualifier)
	if err != nil {
		return nil, fmt.Errorf("could not validate transaction: %w", err)
	}

	txIDs, err := r.index.TransactionsByHeight(*completed.Index)
	if err != nil {
		return nil, fmt.Errorf("could not list block transactions: %w", err)
	}
	var found bool
	for _, txID := range txIDs {
		if txID.String() == txQualifier.Hash {
			found = true
			break
		}
	}
	if !found {
		return nil, failure.UnknownTransaction{
			Hash: txQualifier.Hash,
			Description: failure.NewDescription("transaction not found in given block",
				failure.WithUint64("block_index", *completed.Index),
				failure.WithString("block_hash", completed.Hash),
			),
		}
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
	events, err := r.index.Events(*completed.Index, flow.EventType(deposit), flow.EventType(withdrawal))
	if err != nil {
		return nil, fmt.Errorf("could not get events: %w", err)
	}

	// Convert events to operations and group them by transaction ID.
	var ops []object.Operation
	for _, event := range events {
		// Ignore events that are related to other transactions.
		if event.TransactionID.String() != txQualifier.Hash {
			continue
		}

		op, err := r.convert.EventToOperation(event)
		if errors.Is(err, converter.ErrIrrelevant) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("could not convert event: %w", err)
		}

		ops = append(ops, *op)
	}

	// Set RelatedIDs for all operations for the same transaction.
	for i := range ops {
		for j := range ops {
			if i == j {
				continue
			}

			ops[i].RelatedIDs = append(ops[i].RelatedIDs, ops[j].ID)
		}

	}

	transaction := object.Transaction{
		ID:         txQualifier,
		Operations: ops,
	}

	return &transaction, nil
}
