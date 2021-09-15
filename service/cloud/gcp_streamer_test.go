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

package cloud_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gcloud "cloud.google.com/go/storage"
	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/cloud"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestGCPStreamer_Next(t *testing.T) {
	blockData := mocks.GenericBlockData()
	data, err := cbor.Marshal(blockData)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
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

		buffer := dps.NewDeque()
		buffer.PushFront(blockData)

		streamer := cloud.BaselineStreamer(t, cloud.WithBucket(bucket), cloud.WithBuffer(buffer))

		got, err := streamer.Next()

		require.NoError(t, err)
		assert.Equal(t, blockData, got)
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

		streamer := cloud.BaselineStreamer(t, cloud.WithBucket(bucket))

		_, err = streamer.Next()

		require.Error(t, err)
		assert.ErrorIs(t, err, dps.ErrUnavailable)
	})

	t.Run("downloads records from queue", func(t *testing.T) {
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

		queue := dps.NewDeque()
		queue.PushFront(blockData.Block.ID())

		streamer := cloud.BaselineStreamer(t, cloud.WithBucket(bucket), cloud.WithQueue(queue))

		_, err = streamer.Next()

		require.Error(t, err)
		assert.ErrorIs(t, err, dps.ErrUnavailable)

		select {
		case <-time.After(100 * time.Millisecond):
			t.Fatal("GCP Streamer did not attempt to download record from bucket")
		case <-serverCalled:
		}
	})
}
