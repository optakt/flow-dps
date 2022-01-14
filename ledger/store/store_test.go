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

package store_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/optakt/flow-dps/ledger/store"
	"github.com/optakt/flow-dps/testing/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Eviction(t *testing.T) {

	paths := mocks.GenericLedgerPaths(512)
	payloads := mocks.GenericLedgerPayloads(512)

	t.Run("without concurrency", func(t *testing.T) {
		s, err := store.New(mocks.NoopLogger, store.WithCacheSize(256), store.WithStoragePath(t.TempDir()))
		require.NoError(t, err)

		// Insert all values.
		for i := 0; i < len(paths); i++ {
			h, err := hash.ToHash(paths[i][:])
			require.NoError(t, err)

			err = s.Save(h, payloads[i])
			require.NoError(t, err)
		}

		// Retrieve all values. They should all be accessible, even if only the
		// last 256 should be in the LRU cache.
		for i := 0; i < len(paths); i++ {
			h, err := hash.ToHash(paths[i][:])
			require.NoError(t, err)

			payload, err := s.Retrieve(h)
			require.NoError(t, err)

			assert.Equal(t, payloads[i].Value, payload.Value)
		}
	})

	t.Run("with concurrency", func(t *testing.T) {
		s, err := store.New(mocks.NoopLogger, store.WithCacheSize(256), store.WithStoragePath(t.TempDir()))
		require.NoError(t, err)

		done := make(chan struct{})

		go func() {
			// Insert values randomly until test stops.
			for {
				select {
				case <-done:
					return
				default:
				}

				i := rand.Intn(len(paths))

				h, err := hash.ToHash(paths[i][:])
				require.NoError(t, err)

				err = s.Save(h, payloads[i])
				require.NoError(t, err)
			}
		}()

		var successfulReads int
		go func() {
			// Read values randomly until test stops.
			for {
				select {
				case <-done:
					return
				default:
				}

				i := rand.Intn(len(paths))

				h, err := hash.ToHash(paths[i][:])
				require.NoError(t, err)

				payload, err := s.Retrieve(h)
				if err != nil {
					continue // The entry might not be in the cache yet.
				}

				if assert.Equal(t, payloads[i].Value, payload.Value) {
					successfulReads++
				}
			}
		}()

		<-time.After(5 * time.Second)
		close(done)

		// Make sure that at least some values were read successfully.
		assert.NotZero(t, successfulReads)
	})
}
