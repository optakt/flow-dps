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
	"github.com/optakt/flow-dps/follower/execution"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Source struct {
	log       zerolog.Logger
	execution ExecutionFollower
	queue     *deque.Deque
	data      map[flow.Identifier]item
}

type item struct {
	height       uint64
	commit       flow.StateCommitment
	collections  []*flow.LightCollection
	transactions []*flow.TransactionBody
	results      []*flow.TransactionResult
	events       []flow.Event
}

func NewSource(log zerolog.Logger, execution ExecutionFollower) *Source {

	s := Source{
		log:       log,
		execution: execution,
		queue:     deque.New(),
	}

	return &s
}

func (s *Source) Update() (*ledger.TrieUpdate, error) {

	// If we have updates available in the queue, let's get the oldest one and
	// feed it to the indexer.
	if s.queue.Len() != 0 {
		update := s.queue.PopBack()
		return update.(*ledger.TrieUpdate), nil
	}

	// Get the next block data available from the execution follower and push
	// all of the trie updates into the queue.
	blockData, err := s.execution.Next()
	if err != nil {
		return nil, fmt.Errorf("could not get block data: %w", err)
	}
	for _, update := range blockData.TrieUpdates {
		s.queue.PushBack(update)
	}

	// We should then also index the block data by block ID, so we can provide
	// it to the chain interface as needed.
	err = s.indexBlockData(blockData)
	if err != nil {
		return nil, fmt.Errorf("could not index block data: %w", err)
	}

	return s.Update()
}

// commit => execution follower
// collections => execution follower
// results => execution follower
// events => execution follower
// transactions => execution follower

func (s *Source) Prune(height uint64) {
	for blockID, data := range s.data {
		if data.height <= height {
			delete(s.data, blockID)
		}
	}
}

func (s *Source) Commit(blockID flow.Identifier) (flow.StateCommitment, bool) {
	it, ok := s.data[blockID]
	if !ok {
		return flow.StateCommitment{}, false
	}
	return it.commit, true
}

func (s *Source) Collections(blockID flow.Identifier) ([]*flow.LightCollection, bool) {
	it, ok := s.data[blockID]
	if !ok {
		return nil, false
	}
	return it.collections, true
}

func (s *Source) Transactions(blockID flow.Identifier) ([]*flow.TransactionBody, bool) {
	it, ok := s.data[blockID]
	if !ok {
		return nil, false
	}
	return it.transactions, true
}

func (s *Source) Results(blockID flow.Identifier) ([]*flow.TransactionResult, bool) {
	it, ok := s.data[blockID]
	if !ok {
		return nil, false
	}
	return it.results, true
}

func (s *Source) Events(blockID flow.Identifier) ([]flow.Event, bool) {
	it, ok := s.data[blockID]
	if !ok {
		return nil, false
	}
	return it.events, true
}

func (s *Source) indexBlockData(blockData *execution.BlockData) error {

	// Extract the block ID from the block data.
	blockID := blockData.Header.ID()
	_, ok := s.data[blockID]
	if ok {
		return fmt.Errorf("duplicate block ID for block data (block: %x)", blockID)
	}

	// Extract the light collections from the block data.
	numTransactions := 0
	collections := make([]*flow.LightCollection, 0, len(blockData.Collections))
	for _, collection := range blockData.Collections {
		light := collection.Collection().Light()
		collections = append(collections, &light)
		numTransactions += len(light.Transactions)
	}

	// Extract the transactions from the block data.
	transactions := make([]*flow.TransactionBody, 0, numTransactions)
	for _, collection := range blockData.Collections {
		transactions = append(transactions, collection.Transactions...)
	}

	// Extract the events from the block data.
	events := make([]flow.Event, 0, len(blockData.Events))
	for _, event := range blockData.Events {
		events = append(events, *event)
	}

	// Create and store the item.
	it := item{
		height:       blockData.Header.Height, // needed only for pruning
		commit:       blockData.Commit,
		collections:  collections,
		transactions: transactions,
		results:      blockData.TxResults,
		events:       events,
	}

	s.data[blockID] = it

	return nil
}
