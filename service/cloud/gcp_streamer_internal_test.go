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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	gcloud "cloud.google.com/go/storage"
	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"

	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestNewGCPStreamer(t *testing.T) {
	log := zerolog.Nop()
	bucket := &storage.BucketHandle{}
	limit := uint(42)
	blockIDs := mocks.GenericBlockIDs(4)

	streamer := NewGCPStreamer(
		log,
		bucket,
		WithBufferSize(limit),
		WithCatchupBlocks(blockIDs),
	)

	require.NotNil(t, streamer)
	assert.NotZero(t, streamer.log)
	assert.Equal(t, bucket, streamer.bucket)
	assert.Equal(t, limit, streamer.limit)
	assert.NotNil(t, streamer.queue)
	assert.NotNil(t, streamer.buffer)

	for streamer.queue.Len() > 0 {
		assert.Contains(t, blockIDs, streamer.queue.PopFront())
	}
}

func TestGCPStreamer_OnBlockFinalized(t *testing.T) {
	blockID := mocks.GenericHeader.ID()
	queue := dps.NewDeque()

	streamer := &GCPStreamer{
		log:   zerolog.Nop(),
		queue: queue,
	}

	streamer.OnBlockFinalized(blockID)

	require.Equal(t, 1, queue.Len())
	assert.Equal(t, queue.PopFront(), blockID)
}

func TestGCPStreamer_Next(t *testing.T) {
	record := mocks.GenericRecord()
	data, err := cbor.Marshal(record)
	require.NoError(t, err)

	decOptions := cbor.DecOptions{ExtraReturnErrors: cbor.ExtraDecErrorUnknownField}
	decoder, err := decOptions.DecMode()
	require.NoError(t, err)

	t.Run("returns available record if buffer not empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}))

		client, err := gcloud.NewClient(
			context.Background(),
			option.WithoutAuthentication(),
			option.WithEndpoint(server.URL),
		)
		require.NoError(t, err)
		bucket := client.Bucket("test")

		streamer := &GCPStreamer{
			log:     zerolog.Nop(),
			bucket:  bucket,
			decoder: decoder,
			queue:   dps.NewDeque(),
			buffer:  dps.NewDeque(),
			limit:   999,
		}

		streamer.buffer.PushFront(record)

		got, err := streamer.Next()

		require.NoError(t, err)
		assert.Equal(t, record, got)
	})

	t.Run("returns unavailable when no block data in buffer", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusOK)
		}))

		client, err := gcloud.NewClient(
			context.Background(),
			option.WithoutAuthentication(),
			option.WithEndpoint(server.URL),
		)
		require.NoError(t, err)
		bucket := client.Bucket("test")

		streamer := &GCPStreamer{
			log:     zerolog.Nop(),
			bucket:  bucket,
			decoder: decoder,
			queue:   dps.NewDeque(),
			buffer:  dps.NewDeque(),
			limit:   999,
		}

		_, err = streamer.Next()

		require.Error(t, err)
		assert.ErrorIs(t, err, dps.ErrUnavailable)
	})

	t.Run("downloads records from queue when they are available", func(t *testing.T) {
		serverCalled := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write(data)
			serverCalled <- struct{}{}
		}))

		client, err := gcloud.NewClient(
			context.Background(),
			option.WithoutAuthentication(),
			option.WithEndpoint(server.URL),
		)
		require.NoError(t, err)
		bucket := client.Bucket("test")

		streamer := &GCPStreamer{
			log:     zerolog.Nop(),
			bucket:  bucket,
			decoder: decoder,
			queue:   dps.NewDeque(),
			buffer:  dps.NewDeque(),
			limit:   999,
		}

		streamer.queue.PushFront(record.Block.ID())

		_, err = streamer.Next()

		require.Error(t, err)
		assert.ErrorIs(t, err, dps.ErrUnavailable)

		select {
		case <-time.After(100 * time.Millisecond):
			t.Fatal("GCP Streamer did not attempt to download record from bucket")
		case <-serverCalled:
		}

		assert.Zero(t, streamer.queue.Len())
	})
}
