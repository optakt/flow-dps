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
	"testing"

	"cloud.google.com/go/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewGCPStreamer(t *testing.T) {
	log := zerolog.Nop()
	bucket := &storage.BucketHandle{}
	limit := uint(42)

	streamer := NewGCPStreamer(log, bucket, WithBufferSize(limit))

	require.NotNil(t, streamer)
	assert.Equal(t, bucket, streamer.bucket)
	assert.Equal(t, limit, streamer.limit)
}

func TestNewGCPStreamer_OnBlockFinalized(t *testing.T) {
	blockID := mocks.GenericHeader.ID()
	queue := dps.NewDeque()

	streamer := &GCPStreamer{
		log:   zerolog.Nop(),
		queue: queue,
	}

	streamer.OnBlockFinalized(blockID)

	got := queue.PopFront()

	assert.Equal(t, blockID, got)
}

func BaselineStreamer(t *testing.T, opts ...func(*GCPStreamer)) *GCPStreamer {
	t.Helper()

	stream := GCPStreamer{
		log:    zerolog.Nop(),
		bucket: &storage.BucketHandle{},
		queue:  dps.NewDeque(),
		buffer: dps.NewDeque(),
		limit:  42,
	}

	for _, opt := range opts {
		opt(&stream)
	}

	return &stream
}

func WithBucket(bucket *storage.BucketHandle) func(*GCPStreamer) {
	return func(streamer *GCPStreamer) {
		streamer.bucket = bucket
	}
}

func WithQueue(queue *dps.SafeDeque) func(*GCPStreamer) {
	return func(streamer *GCPStreamer) {
		streamer.queue = queue
	}
}

func WithBuffer(buffer *dps.SafeDeque) func(*GCPStreamer) {
	return func(streamer *GCPStreamer) {
		streamer.buffer = buffer
	}
}
