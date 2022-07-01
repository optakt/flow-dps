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

package cloud

import (
	"context"
	"errors"
	"fmt"
	"github.com/onflow/flow-go/consensus/hotstuff/model"
	"io"
	"sync/atomic"

	"cloud.google.com/go/storage"
	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"
	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-dps/models/dps"
)

// GCPStreamer is a component that downloads block data from a Google Cloud bucket.
// It exposes a callback to be used by the consensus follower to notify the Streamer
// when a new block has been finalized. The streamer will then add that block to the
// queue, which is consumed by downloading the block data for the identifiers it
// contains.
type GCPStreamer struct {
	log     zerolog.Logger
	decoder cbor.DecMode
	bucket  *storage.BucketHandle
	queue   *dps.SafeDeque // queue of block identifiers for next downloads
	buffer  *dps.SafeDeque // queue of downloaded execution data records
	limit   uint           // buffer size limit for downloaded records
	busy    uint32         // used as a guard to avoid concurrent polling
}

// NewGCPStreamer returns a new GCP Streamer using the given bucket and options.
func NewGCPStreamer(log zerolog.Logger, bucket *storage.BucketHandle, options ...Option) *GCPStreamer {

	cfg := DefaultConfig
	for _, option := range options {
		option(&cfg)
	}

	decOptions := cbor.DecOptions{
		ExtraReturnErrors: cbor.ExtraDecErrorUnknownField,
	}
	decoder, err := decOptions.DecMode()
	if err != nil {
		panic(err)
	}

	g := GCPStreamer{
		log:     log.With().Str("component", "gcp_streamer").Logger(),
		decoder: decoder,
		bucket:  bucket,
		queue:   dps.NewDeque(),
		buffer:  dps.NewDeque(),
		limit:   cfg.BufferSize,
		busy:    0,
	}

	for _, blockID := range cfg.CatchupBlocks {
		g.queue.PushFront(blockID)
		g.log.Debug().Hex("block", blockID[:]).Msg("execution record queued for catch-up")
	}

	return &g
}

// OnBlockFinalized is a callback for the Flow consensus follower. It is called
// each time a block is finalized by the Flow consensus algorithm.
func (g *GCPStreamer) OnBlockFinalized(block *model.Block) {
	blockID := block.BlockID
	// We push the block ID to the front of the queue; the streamer will try to
	// download the blocks in a FIFO manner.
	g.queue.PushFront(blockID)

	g.log.Debug().Hex("block", blockID[:]).Msg("execution record queued for download")
}

// Next returns the next available block data. It returns an ErrUnavailable if no block
// data is available at the moment.
func (g *GCPStreamer) Next() (*uploader.BlockData, error) {

	// If we are not polling already, we want to start polling in the
	// background. This will try to fill the buffer up until its limit is
	// reached. It basically means that the cloud streamer will always be
	// downloading if something is available and the execution tracker is asking
	// for the next record.
	go g.poll()

	// If we have nothing in the buffer, we can return the unavailable error,
	// which will cause the mapper logic to go into a wait state and retry a bit
	// later.
	if g.buffer.Len() == 0 {
		g.log.Debug().Msg("buffer empty, no execution record available")
		return nil, dps.ErrUnavailable
	}

	// If we have a record in the buffer, we will just return it. The buffer is
	// concurrency safe, so there is no problem with popping from the back while
	// the poll is pushing new items in the front.
	record := g.buffer.PopBack()
	return record.(*uploader.BlockData), nil
}

func (g *GCPStreamer) poll() {

	// We only call `Next()` sequentially, so there is no need to guard it from
	// concurrent access. However, when the buffer is not empty, we might still
	// be polling for new data in the background when the next call happens. We
	// thus need to ensure that only one poll is executed at the same time. We
	// do this with a simple flag that is set atomically to work like a
	// `TryLock()` on a mutex, which is unfortunately not available in Go, see:
	// https://github.com/golang/go/issues/6123
	if !atomic.CompareAndSwapUint32(&g.busy, 0, 1) {
		return
	}
	defer atomic.StoreUint32(&g.busy, 0)

	// At this point, we try to pull new files from the cloud.
	err := g.download()
	if errors.Is(err, storage.ErrObjectNotExist) {
		g.log.Debug().Msg("next execution record not available, download stopped")
		return
	}
	if err != nil {
		g.log.Error().Err(err).Msg("could not download execution records")
		return
	}
}

func (g *GCPStreamer) download() error {

	for {

		// We only want to retrieve and process files until the buffer is full. We
		// do not need to have a big buffer; we just want to avoid HTTP request
		// latency when the execution follower wants a block record.
		if uint(g.buffer.Len()) >= g.limit {
			g.log.Debug().Uint("limit", g.limit).Msg("buffer full, stopping execution record download")
			return nil
		}

		// We only want to retrieve and process files for blocks that have already
		// been finalized, in the order that they have been finalized. This
		// causes some latency, as we don't download until after a block is
		// finalized, even if the data is available before. However, it seems to
		// be the only way to make sure trie updates are delivered to the mapper
		// in the right order without changing the way uploads work.
		if uint(g.queue.Len()) == 0 {
			g.log.Debug().Msg("queue empty, stopping execution record download")
			return nil
		}

		// Get the name of the file based on the block ID. The file name is
		// made up of the block ID in hex and a `.cbor` extension, see:
		// Maks: "thats correct. In fact the full name is `<blockID>.cbor`"
		// If we encounter an error, such as that the file is not found, we put
		// the block ID back into the queue and return `nil` to stop pulling.
		blockID := g.queue.PopBack().(flow.Identifier)
		name := blockID.String() + ".cbor"
		record, err := g.pullRecord(name)
		if err != nil {
			g.queue.PushBack(blockID)
			return fmt.Errorf("could not pull execution record (name: %s): %w", name, err)
		}

		g.log.Debug().
			Str("name", name).
			Uint64("height", record.Block.Header.Height).
			Hex("block", blockID[:]).
			Msg("pushing execution record into buffer")

		g.buffer.PushFront(record)
	}
}

func (g *GCPStreamer) pullRecord(name string) (*uploader.BlockData, error) {

	object := g.bucket.Object(name)
	reader, err := object.NewReader(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not create object reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read execution record: %w", err)
	}

	var record uploader.BlockData
	err = g.decoder.Unmarshal(data, &record)
	if err != nil {
		return nil, fmt.Errorf("could not decode execution record: %w", err)
	}

	if record.FinalStateCommitment == flow.DummyStateCommitment {
		return nil, fmt.Errorf("execution record contains empty state commitment")
	}

	if record.Block.Header.Height == 0 {
		return nil, fmt.Errorf("execution record contains empty block data")
	}

	return &record, nil
}
