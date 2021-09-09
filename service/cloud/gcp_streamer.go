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
	"time"

	"cloud.google.com/go/storage"
	"github.com/fxamacker/cbor/v2"
	"github.com/gammazero/deque"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type GCPStreamer struct {
	log      zerolog.Logger
	bucket   *storage.BucketHandle
	queue    *deque.Deque
	validate *validator.Validate
	buffer   *deque.Deque
	last     time.Time
	busy     uint32
	wg       *sync.WaitGroup
	limit    uint
}

func NewGCPStreamer(log zerolog.Logger, bucket *storage.BucketHandle) *GCPStreamer {

	g := GCPStreamer{
		log:      log.With().Str("component", "gcp_streamer").Logger(),
		bucket:   bucket,
		queue:    deque.New(),
		validate: validator.New(),
		buffer:   deque.New(),
		last:     time.Time{},
		busy:     0,
		wg:       &sync.WaitGroup{},
		limit:    8,
	}

	return &g
}

func (g *GCPStreamer) OnBlockFinalized(blockID flow.Identifier) {
	g.queue.PushFront(blockID)
}

func (g *GCPStreamer) Next() (*uploader.BlockData, error) {
	g.wg.Add(1)
	go g.poll()

	if g.buffer.Len() == 0 {
		g.log.Debug().Msg("buffer empty, waiting for poll")
		g.wg.Wait()
	}

	if g.buffer.Len() == 0 {
		g.log.Debug().Msg("buffer still empty, no data available")
		return nil, dps.ErrUnavailable
	}

	record := g.buffer.PopBack()
	return record.(*uploader.BlockData), nil
}

func (g *GCPStreamer) poll() {
	defer g.wg.Done()

	if !atomic.CompareAndSwapUint32(&g.busy, 0, 1) {
		return
	}

	defer atomic.StoreUint32(&g.busy, 0)

	err := g.pull()
	if err != nil {
		g.log.Error().Err(err).Msg("could not pull records")
		return
	}
}

func (g *GCPStreamer) pull() error {

	for {

		// We only want to retrieve and process files until the buffer is full. We
		// do not need to have a big buffer, we just want to avoid HTTP request
		// latency when the execution follower wants a block record.
		if uint(g.buffer.Len()) >= g.limit {
			g.log.Debug().Uint("limit", g.limit).Msg("buffer full, finishing pull")
			return nil
		}

		// We only want to retrieve and process files for blocks that have already
		// been finalized, in the order that they have been finalized. This creates
		// some latency, but it's currently the only way we can ensure that trie
		// updates are delivered in the right order.
		if uint(g.queue.Len()) == 0 {
			g.log.Debug().Msg("queue empty, finishing pull")
			return nil
		}

		// Get the name of the file based on the block ID.
		blockID := g.queue.PopBack().(flow.Identifier)
		name := blockID.String() + ".block"
		record, err := g.pullRecord(name)
		if err != nil {
			return fmt.Errorf("could not pull record object (name: %s): %w", name, err)
		}

		g.log.Debug().Str("name", name).Msg("pushing record object into buffer")

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

	err = g.validate.Struct(record)
	if err != nil {
		return nil, fmt.Errorf("could not validate record: %w", err)
	}

	return &record, nil
}
