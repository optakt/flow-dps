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

package gcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"

	"github.com/optakt/flow-dps/testing/mocks"
)

func TestReader_Read(t *testing.T) {
	blockID := mocks.GenericIdentifier(0)

	// Set up fake GCP server for testing, which always returns no error.
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, err := rw.Write(mocks.GenericBytes)
		require.NoError(t, err)

		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Make the GCP Storage client use that server.
	url, err := url.Parse(server.URL)
	require.NoError(t, err)
	_ = os.Setenv("STORAGE_EMULATOR_HOST", url.Host)
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithoutAuthentication(), option.WithHTTPClient(server.Client()))
	require.NoError(t, err)

	bucket := client.Bucket("my-bucket")

	fileName := fmt.Sprintf("%x.cbor", blockID[:])

	t.Run("nominal case", func(t *testing.T) {
		downloader := &Downloader{
			bucket: bucket,
			cache: map[string]time.Time{
				fileName: time.Now(),
			},
		}

		got, err := downloader.Read(blockID)

		assert.NoError(t, err)
		assert.Equal(t, mocks.GenericBytes, got)
	})

	t.Run("handles item missing from cache", func(t *testing.T) {
		downloader := &Downloader{
			bucket: bucket,
			cache:  map[string]time.Time{},
		}

		_, err = downloader.Read(blockID)

		assert.Error(t, err)
	})
}
