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
	"github.com/gammazero/deque"
	"github.com/rs/zerolog"
	"google.golang.org/api/iterator"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/follower"
)

type GCPStream struct {
	log    zerolog.Logger
	bucket *storage.BucketHandle
	codec  dps.Codec
	buffer *deque.Deque
	last   time.Time
	wg     *sync.WaitGroup
	limit  uint
}

func NewGCPStream(log zerolog.Logger, bucket *storage.BucketHandle, codec dps.Codec) *GCPStream {

	g := GCPStream{
		log:    log,
		bucket: bucket,
		codec:  codec,
		buffer: deque.New(),
		last:   time.Time{},
		wg:     &sync.WaitGroup{},
		limit:  8,
	}

	return &g
}

func (g *GCPStream) Next() (*follower.Record, error) {
	g.wg.Add(1)
	defer g.poll()

	if g.buffer.Len() == 0 {
		g.wg.Wait()
	}

	if g.buffer.Len() == 0 {
		return nil, dps.ErrUnavailable
	}

	record := g.buffer.PopBack()
	return record.(*follower.Record), nil
}

func (g *GCPStream) poll() {
	defer g.wg.Done()

	err := g.pull()
	if err != nil {
		g.log.Error().Err(err).Msg("could not pull records")
	}
}

func (g *GCPStream) pull() error {

	// We only want to retrieve and process files until the buffer is full. We
	// do not need to have a big buffer, we just want to avoid HTTP request
	// latency when the execution follower wants a block record.
	if uint(g.buffer.Len()) >= g.limit {
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
		if !object.Created.After(g.last) {
			// This filters out all objects we already processed.
			continue
		}
		objects = append(objects, object)
	}

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

		g.buffer.PushFront(record)

		if uint(g.buffer.Len()) >= g.limit {
			break
		}
	}

	return nil
}

func (g *GCPStream) pullRecord(name string) (*follower.Record, error) {

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

	var record follower.Record
	err = g.codec.Decode(data, &record)
	if err != nil {
		return nil, fmt.Errorf("could not decode record: %w", err)
	}

	return &record, nil
}
