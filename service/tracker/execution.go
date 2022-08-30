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

package tracker

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/gammazero/deque"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"
)

// Execution is the DPS execution follower, which keeps track of updates to the
// execution state. It retrieves block records (block data updates) from a
// streamer and extracts the trie updates for consumers. It also makes the rest
// of the block record data available for external consumers by block ID.
type Execution struct {
	log     zerolog.Logger
	queue   *deque.Deque
	stream  RecordStreamer
	records map[flow.Identifier]*uploader.BlockData
}

// NewExecution creates a new DPS execution follower, relying on the provided
// stream of block records (block data updates).
func NewExecution(log zerolog.Logger, db *badger.DB, stream RecordStreamer) (*Execution, error) {

	// The root block does not have a record that we can pull from the cloud
	// stream of execution data. We thus construct it by getting the root block
	// data from the DB directly.
	var height uint64
	err := db.View(operation.RetrieveRootHeight(&height))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve root height: %w", err)
	}
	var blockID flow.Identifier
	err = db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up root block: %w", err)
	}
	var header flow.Header
	err = db.View(operation.RetrieveHeader(blockID, &header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve root header: %w", err)
	}
	var sealID flow.Identifier
	err = db.View(operation.LookupBySealedBlockID(blockID, &sealID))
	if err != nil {
		return nil, fmt.Errorf("could not look up root seal: %w", err)
	}
	var seal flow.Seal
	err = db.View(operation.RetrieveSeal(sealID, &seal))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve root seal: %w", err)
	}

	e := Execution{
		log:     log.With().Str("component", "execution_tracker").Logger(),
		stream:  stream,
		queue:   deque.New(),
		records: make(map[flow.Identifier]*uploader.BlockData),
	}

	payload := flow.Payload{
		Guarantees: nil,
		Seals:      nil,
		Receipts:   nil,
		Results:    nil,
	}

	block := flow.Block{
		Header:  &header,
		Payload: &payload,
	}

	record := uploader.BlockData{
		Block:                &block,
		Collections:          nil, // no collections
		TxResults:            nil, // no transaction results
		Events:               nil, // no events
		TrieUpdates:          nil, // no trie updates
		FinalStateCommitment: seal.FinalState,
	}

	e.records[blockID] = &record

	return &e, nil
}

// Update provides the next trie update from the stream of block records. Trie
// updates are returned sequentially without regard for the boundary between
// blocks.
func (e *Execution) Update() (*ledger.TrieUpdate, error) {

	// If we have updates available in the queue, let's get the oldest one and
	// feed it to the indexer.
	if e.queue.Len() != 0 {
		update := e.queue.PopBack()
		return update.(*ledger.TrieUpdate), nil
	}

	// We should then also index the block data by block ID, so we can provide
	// it to the chain interface as needed.
	err := e.processNext()
	if err != nil {
		return nil, fmt.Errorf("could not process next execution record: %w", err)
	}

	// This is a recursive function call. It allows us to skip past blocks which
	// don't contain trie updates. It will stop recursing once a block has
	// trie updates or when no more blocks are available from the streamer.
	return e.Update()
}

// Record returns the block record for the given block ID, if it is available.
// Once a block record is returned, all block records at a height lower than
// the height of the returned record are purged from the cache.
func (e *Execution) Record(blockID flow.Identifier) (*uploader.BlockData, error) {

	// If we have the block available in the cache, let's feed it to the
	// consumer.
	record, ok := e.records[blockID]
	if ok {
		e.purge(record.Block.Header.Height)
		return record, nil
	}

	// Get the next block data available from the execution follower and process
	// it appropriately. This will wrap an unavailable error if we don't get
	// the next one from the cloud reader.
	err := e.processNext()
	if err != nil {
		return nil, fmt.Errorf("could not process next execution record: %w", err)
	}

	// This is a recursive function call. It allows us to keep reading block
	// records from the cloud streamer until we find the block we are looking
	// for, or until we receive an unavailable error that we propagate up.
	return e.Record(blockID)
}

func (e *Execution) processNext() error {

	// Get the next block execution record available from the cloud streamer.
	record, err := e.stream.Next()
	if err != nil {
		return fmt.Errorf("could not read next execution record: %w", err)
	}

	// Check if we already processed a block with this ID recently. This should
	// be idempotent, but we should be aware if something like this happens.
	blockID := record.Block.Header.ID()
	_, ok := e.records[blockID]
	if ok {
		return fmt.Errorf("duplicate execution record (block: %x)", blockID)
	}

	// Dump the block execution record into our cache and push all trie updates
	// into our update queue.
	e.records[blockID] = record
	for _, update := range record.TrieUpdates {

		// The Flow execution node includes `nil` updates in the slice, instead
		// of empty updates. We could fix this here, but we don't have the root
		// hash to apply against, so we just skip.
		if update == nil {
			continue
		}

		e.queue.PushFront(update)
	}

	e.log.Debug().
		Hex("block", blockID[:]).
		Int("updates", len(record.TrieUpdates)).
		Msg("next execution record processed")

	return nil
}

// purge deletes all records that are below the specified height threshold.
func (e *Execution) purge(threshold uint64) {
	for blockID, record := range e.records {
		if record.Block.Header.Height < threshold {
			delete(e.records, blockID)
		}
	}
}
