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
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"cloud.google.com/go/storage"
	"github.com/fxamacker/cbor/v2"
	"github.com/gammazero/deque"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type GCPStreamer struct {
	log    zerolog.Logger
	bucket *storage.BucketHandle
	queue  *deque.Deque // queue of block identifiers for next downloads
	buffer *deque.Deque // queue of downloaded execution data records
	busy   uint32       // used as a guard to avoid concurrent polling
	wg     *sync.WaitGroup
	limit  uint
}

func NewGCPStreamer(log zerolog.Logger, bucket *storage.BucketHandle) *GCPStreamer {

	g := GCPStreamer{
		log:    log.With().Str("component", "gcp_streamer").Logger(),
		bucket: bucket,
		queue:  deque.New(),
		buffer: deque.New(),
		busy:   0,
		wg:     &sync.WaitGroup{},
		limit:  8,
	}

	return &g
}

// OnBlockFinalized is a callback for the Flow consensus follower. It is called
// each time a block is finalized by the Flow consensus algorithm.
func (g *GCPStreamer) OnBlockFinalized(blockID flow.Identifier) {

	// We push the block ID to the front of the queue; the streamer will try to
	// download the blocks in a FIFO manner.
	g.queue.PushFront(blockID)
}

func (g *GCPStreamer) Next() (*uploader.BlockData, error) {

	// We want to be able to wait for a poll to successfully complete before
	// returning, so we add to the waiting group and start the polling in the
	// background.
	g.wg.Add(1)
	go g.poll()

	// However, in case we already have items in the buffer, we prefer to return
	// immediately and complete the polling in the background, so the buffer is
	// filled again for the next request. We thus only wait on the wait group if
	// the buffer is empty.
	if g.buffer.Len() == 0 {
		g.log.Debug().Msg("buffer empty, waiting for poll")
		g.wg.Wait()
	}

	// At this point, we waited for polling to finish. If the buffer is still
	// empty, it means there is no new data available on the cloud, and we can
	// return the corresponding error.
	if g.buffer.Len() == 0 {
		g.log.Debug().Msg("buffer still empty, no data available")
		return nil, dps.ErrUnavailable
	}

	// When we get here, we either had a full buffer before finishing polling,
	// or the buffer was filled up again by the polling. We can thus pop an item
	// from the buffer and return it.
	record := g.buffer.PopBack()
	return record.(*uploader.BlockData), nil
}

func (g *GCPStreamer) poll() {

	// We defer the call to done, so that the consumer is notified of the
	// finished poll in case it is waiting.
	defer g.wg.Done()

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
	err := g.pull()
	if err != nil {
		g.log.Error().Err(err).Msg("could not pull records")
		return
	}
}

func (g *GCPStreamer) pull() error {

	for {

		// We only want to retrieve and process files until the buffer is full. We
		// do not need to have a big buffer; we just want to avoid HTTP request
		// latency when the execution follower wants a block record.
		if uint(g.buffer.Len()) >= g.limit {
			g.log.Debug().Uint("limit", g.limit).Msg("buffer full, finishing pull")
			return nil
		}

		// We only want to retrieve and process files for blocks that have already
		// been finalized, in the order that they have been finalized. This
		// causes some latency, as we don't download until after a block is
		// finalized, even if the data is available before. However, it seems to
		// be the only way to make sure trie updates are delivered to the mapper
		// in the right order without changing the way uploads work.
		if uint(g.queue.Len()) == 0 {
			g.log.Debug().Msg("queue empty, finishing pull")
			return nil
		}

		// Get the name of the file based on the block ID. The file name is
		// made up of the block ID in hex and a `.cbor` extension, see:
		// Maks: "thats correct. In fact the full name is `<blockID>.cbor`"
		blockID := g.queue.PopBack().(flow.Identifier)
		name := blockID.String() + ".cbor"
		record, err := g.pullRecord(name)
		if err != nil {
			return fmt.Errorf("could not pull record object (name: %s): %w", name, err)
		}

		g.log.Debug().
			Str("name", name).
			Uint64("height", record.Block.Header.Height).
			Hex("block", blockID[:]).
			Msg("pushing record object into buffer")

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
		return nil, fmt.Errorf("could not read object data: %w", err)
	}

	var record uploader.BlockData
	err = cbor.Unmarshal(data, &record)
	if err != nil {
		return nil, fmt.Errorf("could not decode record: %w", err)
	}

	if record.FinalStateCommitment == flow.DummyStateCommitment {
		return nil, fmt.Errorf("record does not contain state commitment")
	}

	if record.Block.Header.Height == 0 {
		return nil, fmt.Errorf("record does not contain block data")
	}

	return &record, nil
}
