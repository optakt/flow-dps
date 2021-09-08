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

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
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
func NewExecution(log zerolog.Logger, stream RecordStreamer) *Execution {

	s := Execution{
		log:     log,
		stream:  stream,
		queue:   deque.New(),
		records: make(map[flow.Identifier]*uploader.BlockData),
	}

	return &s
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

	// Get the next block data available from the execution follower and push
	// all of the trie updates into the queue.
	record, err := e.stream.Next()
	if err != nil {
		return nil, fmt.Errorf("could not read record: %w", err)
	}
	for _, update := range record.TrieUpdates {
		e.queue.PushFront(update)
	}

	// We should then also index the block data by block ID, so we can provide
	// it to the chain interface as needed.
	err = e.processRecord(record)
	if err != nil {
		return nil, fmt.Errorf("could not index block record: %w", err)
	}

	// This is a recursive function call. It allows us to skip past blocks which
	// don't contain trie updates. It will stop recursing once a block has
	// trie updates or when no more blocks are available from the streamer.
	return e.Update()
}

// Record returns the block record for the given block ID, if it is available.
// Once a block record is returned, all block records at a height lower than
// the height of the returned record are purged from the cache.
func (e *Execution) Record(blockID flow.Identifier) (*uploader.BlockData, bool) {
	record, ok := e.records[blockID]
	if !ok {
		return nil, false
	}
	e.purge(record.Block.Header.Height)
	return record, true
}

func (e *Execution) processRecord(record *uploader.BlockData) error {

	// Extract the block ID from the block data.
	blockID := record.Block.Header.ID()
	_, ok := e.records[blockID]
	if ok {
		return fmt.Errorf("duplicate block record (block: %x)", blockID)
	}

	e.records[blockID] = record

	return nil
}

func (e *Execution) purge(threshold uint64) {
	for blockID, record := range e.records {
		if record.Block.Header.Height < threshold {
			delete(e.records, blockID)
		}
	}
}
