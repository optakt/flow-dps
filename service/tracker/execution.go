package tracker

import (
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/gammazero/deque"
	"github.com/onflow/flow-go/engine/common/rpc/convert"
	"github.com/onflow/flow-go/engine/execution/ingestion/uploader"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"
	access "github.com/onflow/flow/protobuf/go/flow/executiondata"
	"github.com/rs/zerolog"
)

// Execution is the DPS execution follower, which keeps track of updates to the
// execution state. It retrieves block records (block data updates) from a
// streamer and extracts the trie updates for consumers. It also makes the rest
// of the block record data available for external consumers by block ID.
type Execution struct {
	log        zerolog.Logger
	queue      *deque.Deque
	stream     RecordStreamer
	records    map[flow.Identifier]*uploader.BlockData
	execClient access.ExecutionDataAPIClient
	chain      flow.Chain
}

// NewExecution creates a new DPS execution follower, relying on the provided
// stream of block records (block data updates).
func NewExecution(
	log zerolog.Logger,
	db *badger.DB,
	stream RecordStreamer,
	execClient access.ExecutionDataAPIClient,
	chain flow.Chain,
) (*Execution, error) {

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
	err = db.View(operation.LookupLatestSealAtBlock(blockID, &sealID))
	if err != nil {
		return nil, fmt.Errorf("could not look up root seal: %w", err)
	}
	var seal flow.Seal
	err = db.View(operation.RetrieveSeal(sealID, &seal))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve root seal: %w", err)
	}

	e := Execution{
		log:        log.With().Str("component", "execution_tracker").Logger(),
		stream:     stream,
		queue:      deque.New(),
		records:    make(map[flow.Identifier]*uploader.BlockData),
		execClient: execClient,
		chain:      chain,
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

// AllUpdates provides all trie updates in the execution record of the next block in the queue
func (e *Execution) AllUpdates() ([]*ledger.TrieUpdate, error) {

	// If we have updates available in the queue, let's get the oldest one and
	// feed it to the indexer.
	if e.queue.Len() != 0 {
		update := e.queue.PopBack()
		return update.([]*ledger.TrieUpdate), nil
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
	return e.AllUpdates()
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
	if e.execClient != nil {
		e.log.Debug().Hex("block_id", blockID[:]).Msg("fetching updates from data sync")
		e.log.Debug().Int("updates", len(record.TrieUpdates)).Msg("got trie updates from GCP")
		// get Trie updates from exec data sync
		req := &access.GetExecutionDataByBlockIDRequest{BlockId: blockID[:]}
		res, err := e.execClient.GetExecutionDataByBlockID(context.Background(), req)
		if err != nil {
			return fmt.Errorf("could not get execution data from access node: %w", err)
		}

		execTrieUpdates := make(map[string]bool, 0)
		// collect updates before pushing!
		for _, chunk := range res.GetBlockExecutionData().ChunkExecutionData {
			convertedChunk, err := convert.MessageToChunkExecutionData(chunk, e.chain)
			if err != nil {
				return fmt.Errorf("unable to convert execution data chunk : %w", err)
			}
			if convertedChunk.TrieUpdate != nil {
				execTrieUpdates[convertedChunk.TrieUpdate.String()] = true
				e.log.Debug().Str("source", "exec sync").Str("update", convertedChunk.TrieUpdate.String()).Msg("")
			}
		}
		// hash search for matching update
		for _, gcpUpdate := range record.TrieUpdates {
			if gcpUpdate != nil && !gcpUpdate.IsEmpty() {
				e.log.Debug().Str("source", "gcp").Str("update", gcpUpdate.String()).Msg("comparing update")
				if !execTrieUpdates[gcpUpdate.String()] {
					return fmt.Errorf("got %s mismatching trie updates between exec sync and GCP", gcpUpdate.String())
				}
			}
		}
		// alternatively, we can just use the trie updates we collected from data sync!!
		// e.queue.PushFront(execTrieUpdates)
	}
	e.queue.PushFront(record.TrieUpdates)

	e.log.Info().
		Hex("block", blockID[:]).
		Uint64("height", record.Block.Header.Height).
		Int("trie_updates", len(record.TrieUpdates)).
		Msg("next execution record processed")

	return nil
}

// purge deletes all records that are below the specified height threshold.
func (e *Execution) purge(threshold uint64) {
	purged := uint64(0)
	for blockID, record := range e.records {
		if record.Block.Header.Height < threshold {
			delete(e.records, blockID)
			purged++
		}
	}
	e.log.Info().Uint64("threshold", threshold).Uint64("purged", purged).Msgf("finish purge")
}
