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

package follower

import (
	"fmt"

	"github.com/gammazero/deque"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Execution struct {
	log    zerolog.Logger
	reader CloudReader
	queue  *deque.Deque
	data   map[flow.Identifier]exeItem
}

type exeItem struct {
	height       uint64
	commit       flow.StateCommitment
	collections  []*flow.LightCollection
	transactions []*flow.TransactionBody
	results      []*flow.TransactionResult
	events       []flow.Event
}

func NewExecution(log zerolog.Logger, reader CloudReader) *Execution {

	s := Execution{
		log:    log,
		reader: reader,
		queue:  deque.New(),
		data:   make(map[flow.Identifier]exeItem),
	}

	return &s
}

func (e *Execution) Update() (*ledger.TrieUpdate, error) {

	// If we have updates available in the queue, let's get the oldest one and
	// feed it to the indexer.
	if e.queue.Len() != 0 {
		update := e.queue.PopBack()
		return update.(*ledger.TrieUpdate), nil
	}

	// Get the next block data available from the execution follower and push
	// all of the trie updates into the queue.
	record, err := e.reader.NextRecord()
	if err != nil {
		return nil, fmt.Errorf("could not get block data: %w", err)
	}
	for _, update := range record.TrieUpdates {
		e.queue.PushBack(update)
	}

	// We should then also index the block data by block ID, so we can provide
	// it to the chain interface as needed.
	err = e.indexRecord(record)
	if err != nil {
		return nil, fmt.Errorf("could not index block data: %w", err)
	}

	return e.Update()
}

func (e *Execution) Commit(blockID flow.Identifier) (flow.StateCommitment, bool) {
	it, ok := e.data[blockID]
	if !ok {
		return flow.DummyStateCommitment, false
	}
	return it.commit, true
}

func (e *Execution) Collections(blockID flow.Identifier) ([]*flow.LightCollection, bool) {
	it, ok := e.data[blockID]
	if !ok {
		return nil, false
	}
	return it.collections, true
}

func (e *Execution) Transactions(blockID flow.Identifier) ([]*flow.TransactionBody, bool) {
	it, ok := e.data[blockID]
	if !ok {
		return nil, false
	}
	return it.transactions, true
}

func (e *Execution) Results(blockID flow.Identifier) ([]*flow.TransactionResult, bool) {
	it, ok := e.data[blockID]
	if !ok {
		return nil, false
	}
	return it.results, true
}

func (e *Execution) Events(blockID flow.Identifier) ([]flow.Event, bool) {
	it, ok := e.data[blockID]
	if !ok {
		return nil, false
	}
	return it.events, true
}

func (e *Execution) indexRecord(record *Record) error {

	// Extract the block ID from the block data.
	blockID := record.Header.ID()
	_, ok := e.data[blockID]
	if ok {
		return fmt.Errorf("execution data duplicate (block: %x)", blockID)
	}

	// Extract the light collections from the block data.
	numTransactions := 0
	collections := make([]*flow.LightCollection, 0, len(record.Collections))
	for _, collection := range record.Collections {
		light := collection.Collection().Light()
		collections = append(collections, &light)
		numTransactions += len(light.Transactions)
	}

	// Extract the transactions from the block data.
	transactions := make([]*flow.TransactionBody, 0, numTransactions)
	for _, collection := range record.Collections {
		transactions = append(transactions, collection.Transactions...)
	}

	// Extract the events from the block data.
	events := make([]flow.Event, 0, len(record.Events))
	for _, event := range record.Events {
		events = append(events, *event)
	}

	// Create and store the item.
	it := exeItem{
		height:       record.Header.Height, // needed only for pruning
		commit:       record.Commit,
		collections:  collections,
		transactions: transactions,
		results:      record.TxResults,
		events:       events,
	}

	e.data[blockID] = it

	return nil
}
