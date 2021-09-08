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
	"sort"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/fxamacker/cbor/v2"
	"github.com/gammazero/deque"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"google.golang.org/api/iterator"

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"

	"github.com/optakt/flow-dps/models/dps"
)

type GCPStreamer struct {
	log      zerolog.Logger
	bucket   *storage.BucketHandle
	validate *validator.Validate
	buffer   *deque.Deque
	last     time.Time
	wg       *sync.WaitGroup
	limit    uint
}

func NewGCPStreamer(log zerolog.Logger, bucket *storage.BucketHandle) *GCPStreamer {

	g := GCPStreamer{
		log:      log.With().Str("component", "gcp_streamer").Logger(),
		bucket:   bucket,
		validate: validator.New(),
		buffer:   deque.New(),
		last:     time.Time{},
		wg:       &sync.WaitGroup{},
		limit:    8,
	}

	return &g
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

	err := g.pull()
	if err != nil {
		g.log.Error().Err(err).Msg("could not pull records")
		return
	}
}

func (g *GCPStreamer) pull() error {

	// We only want to retrieve and process files until the buffer is full. We
	// do not need to have a big buffer, we just want to avoid HTTP request
	// latency when the execution follower wants a block record.
	if uint(g.buffer.Len()) >= g.limit {
		g.log.Debug().Uint("limit", g.limit).Msg("buffer full, not executing pull")
		return nil
	}

	// Retrieve the attributes for all files in the bucket.
	// NOTE: The used attributes are always valid, no need to check error.
	// TODO: We should redo the storage naming so that we can use the prefix to
	// limit the date selection on query. Otherwise, we will end up with a huge
	// iterator every time we pull once there are many block records.
	// => https://github.com/optakt/flow-dps/issues/396
	query := &storage.Query{}
	_ = query.SetAttrSelection([]string{"Name", "Created"})
	it := g.bucket.Objects(context.Background(), query)
	var objects []*storage.ObjectAttrs
	for {
		object, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read next object: %w", err)
		}
		if object.Created.Before(g.last) {
			// This filters out all objects we already processed.
			continue
		}
		objects = append(objects, object)
	}

	g.log.Debug().Int("objects", len(objects)).Msg("polled new block record objects")

	// Now, we sort the objects by creation time to make sure we process the
	// oldest ones first.
	sort.Slice(objects, func(i int, j int) bool {
		return objects[i].Created.Before(objects[j].Created)
	})

	// Then, we read and decode the next file until the buffer is full.
	for _, object := range objects {

		record, err := g.pullRecord(object.Name)
		if err != nil {
			return fmt.Errorf("could not pull record (name: %s): %w", object.Name, err)
		}

		g.last = object.Created

		g.log.Debug().Time("created", object.Created).Msg("pushing block record into buffer")

		g.buffer.PushFront(record)
		if uint(g.buffer.Len()) >= g.limit {
			break
		}
	}

	return nil
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
