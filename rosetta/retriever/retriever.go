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
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"
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

func (r *Retriever) Balances(rosBlockID identifier.Block, rosAccountID identifier.Account, rosCurrencies []identifier.Currency) (identifier.Block, []object.Amount, error) {

	// Run validation on the block qualifier. This also fills in missing fields, where possible.
	completed, err := r.validate.Block(rosBlockID)
	if err != nil {
		return identifier.Block{}, nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Run validation on the account qualifier. This uses the chain ID to check the
	// address validation.
	err = r.validate.Account(rosAccountID)
	if err != nil {
		return identifier.Block{}, nil, fmt.Errorf("could not validate account: %w", err)
	}

	// Run validation on the currency qualifiers. This checks basically if we know the
	// currency and if it has the correct decimals set, if they are set.
	for idx, currency := range rosCurrencies {
		completeCurrency, err := r.validate.Currency(currency)
		if err != nil {
			return identifier.Block{}, nil, fmt.Errorf("could not validate currency: %w", err)
		}
		rosCurrencies[idx] = completeCurrency
	}

	// Get the Cadence value that is the result of the script execution.
	amounts := make([]object.Amount, 0, len(rosCurrencies))
	address := cadence.NewAddress(flow.HexToAddress(rosAccountID.Address))
	for _, currency := range rosCurrencies {
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

func (r *Retriever) Block(rosBlockID identifier.Block) (*object.Block, []identifier.Transaction, error) {

	// Run validation on the block ID. This also fills in missing information.
	completed, err := r.validate.Block(rosBlockID)
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

	// Get all transaction IDs for this height.
	txIDs, err := r.index.TransactionsByHeight(*completed.Index)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get transaction by height: %w", err)
	}

	// Go over all the transaction IDs and create the related Rosetta transaction
	// until we hit the limit, at which point we just add the identifier.
	var blockTransactions []*object.Transaction
	var extraTransactions []identifier.Transaction
	for index, txID := range txIDs {
		if index > int(r.cfg.TransactionLimit) {
			extraTransactions = append(extraTransactions, identifier.Transaction{Hash: txID.String()})
			continue
		}
		ops, err := r.operations(txID, events)
		if err != nil {
			return nil, nil, fmt.Errorf("could not get operations: %w", err)
		}
		rosTx := object.Transaction{
			ID:         identifier.Transaction{Hash: txID.String()},
			Operations: ops,
		}
		blockTransactions = append(blockTransactions, &rosTx)
	}

	// Rosetta spec notes that for genesis block, it is recommended to use the
	// genesis block identifier also for the parent block identifier.
	// See https://www.rosetta-api.org/docs/common_mistakes.html#malformed-genesis-block
	// We thus initialize the parent as the current block, and if the header is
	// not the root block, we use it's actual parent.
	var parent identifier.Block
	first, err := r.index.First()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get first block index: %w", err)
	}
	if header.Height == first {
		parent = completed
	} else {
		height := *completed.Index - 1
		parent.Index = &height
		parent.Hash = header.ParentID.String()
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

func (r *Retriever) Transaction(rosBlockID identifier.Block, rosTxID identifier.Transaction) (*object.Transaction, error) {

	// Run validation on the block qualifier. This also fills in missing information.
	completed, err := r.validate.Block(rosBlockID)
	if err != nil {
		return nil, fmt.Errorf("could not validate block: %w", err)
	}

	// Run validation on the transaction qualifier. This should never fail, as we
	// already check the length, but let's run it anyway.
	err = r.validate.Transaction(rosTxID)
	if err != nil {
		return nil, fmt.Errorf("could not validate transaction: %w", err)
	}

	txIDs, err := r.index.TransactionsByHeight(*completed.Index)
	if err != nil {
		return nil, fmt.Errorf("could not list block transactions: %w", err)
	}
	var found bool
	for _, txID := range txIDs {
		if txID.String() == rosTxID.Hash {
			found = true
			break
		}
	}
	if !found {
		return nil, failure.UnknownTransaction{
			Hash: rosTxID.Hash,
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
	txID, err := flow.HexStringToIdentifier(rosTxID.Hash)
	if err != nil {
		return nil, fmt.Errorf("could not convert Rosetta transaction hash to Flow identifier: %w", err)
	}
	ops, err := r.operations(txID, events)
	if err != nil {
		return nil, fmt.Errorf("could not convert events to operations: %w", err)
	}

	transaction := object.Transaction{
		ID:         rosTxID,
		Operations: ops,
	}

	return &transaction, nil
}

// operations allows us to extract the operations for a transaction ID by using the given list of
// events. In general, we retrieve all events for the block in question, so those should be passed in order to avoid
// querying events for each transaction in a block.
func (r *Retriever) operations(txID flow.Identifier, events []flow.Event) ([]*object.Operation, error) {

	// These are the currently supported event types. The order here has to be kept the same so that we can keep
	// deterministic operation indices, which is a requirement of the Rosetta API specification.
	deposit, err := r.generator.TokensDeposited(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate deposit event type: %w", err)
	}
	withdrawal, err := r.generator.TokensWithdrawn(dps.FlowSymbol)
	if err != nil {
		return nil, fmt.Errorf("could not generate withdrawal event type: %w", err)
	}
	priorities := map[string]uint{
		deposit:    1,
		withdrawal: 2,
	}

	// We then start by filtering out all events that don't have the right transaction
	// ID or which are not a supported type. Afterwards, we sort them by priority,
	// which  will make sure that we keep a deterministic index order for operations.
	filtered := make([]flow.Event, 0, len(events))
	for _, event := range events {
		if event.TransactionID != txID {
			continue
		}
		_, ok := priorities[string(event.Type)]
		if !ok {
			continue
		}
		filtered = append(filtered, event)
	}
	sort.Slice(filtered, func(i int, j int) bool {
		return priorities[string(events[i].Type)] < priorities[string(events[j].Type)]
	})

	// Now we can convert each event to an operation, as they are both filtered for
	// only supported ones and properly ordered.
	ops := make([]*object.Operation, 0, len(filtered))
	for index, event := range filtered {
		op, err := r.convert.EventToOperation(uint(index), event)
		if err != nil {
			return nil, fmt.Errorf("could not convert event to operation (tx: %s, type: %s): %w", event.TransactionID, event.Type, err)
		}
		ops = append(ops, op)
	}

	// Finally, we want the operations within a transaction to be related to each
	// other.
	for _, op1 := range ops {
		for _, op2 := range ops {
			if op1.ID.Index == op2.ID.Index {
				continue
			}
			op1.RelatedIDs = append(op1.RelatedIDs, op2.ID)
		}
	}

	return ops, nil
}
