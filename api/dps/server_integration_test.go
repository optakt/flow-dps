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

//+build integration

package dps_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/service/storage"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestServer_GetRegisterValues(t *testing.T) {
	paths := mocks.GenericLedgerPaths(4)
	values := mocks.GenericLedgerValues(4)
	payloads := mocks.GenericLedgerPayloads(4)
	height := mocks.GenericHeight

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		codec, err := zbor.NewCodec()
		require.NoError(t, err)

		db := helpers.InMemoryDB(t)

		storage := storage.New(codec)
		reader := index.NewReader(db, storage)
		writer := index.NewWriter(db, storage)

		// Insert mock data in database.
		require.NoError(t, writer.First(height))
		require.NoError(t, writer.Last(height))
		require.NoError(t, writer.Payloads(height, paths, payloads))
		require.NoError(t, writer.Close())

		server := dps.NewServer(reader, codec)

		req := &dps.GetRegisterValuesRequest{
			Height: height,
			Paths:  convert.PathsToBytes(paths),
		}
		resp, err := server.GetRegisterValues(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, height, resp.Height)
		assert.Equal(t, convert.ValuesToBytes(values), resp.Values)
		assert.Equal(t, convert.PathsToBytes(paths), resp.Paths)
	})

	t.Run("handles conversion error", func(t *testing.T) {
		t.Parallel()

		codec, err := zbor.NewCodec()
		require.NoError(t, err)

		db := helpers.InMemoryDB(t)

		storage := storage.New(codec)
		reader := index.NewReader(db, storage)
		writer := index.NewWriter(db, storage)

		// Insert mock data in database.
		require.NoError(t, writer.First(height))
		require.NoError(t, writer.Last(height))
		require.NoError(t, writer.Payloads(height, paths, payloads))
		require.NoError(t, writer.Close())

		server := dps.NewServer(reader, codec)

		req := &dps.GetRegisterValuesRequest{
			Height: height,
			Paths:  [][]byte{mocks.GenericBytes},
		}
		_, err = server.GetRegisterValues(context.Background(), req)

		assert.Error(t, err)
	})

	t.Run("handles indexer failure on Values", func(t *testing.T) {
		t.Parallel()

		codec, err := zbor.NewCodec()
		require.NoError(t, err)

		db := helpers.InMemoryDB(t)

		storage := storage.New(codec)
		reader := index.NewReader(db, storage)

		// No data is written in the database, so the index should fail to retrieve payloads.

		server := dps.NewServer(reader, codec)

		req := &dps.GetRegisterValuesRequest{
			Height: height,
			Paths:  [][]byte{mocks.GenericBytes},
		}
		_, err = server.GetRegisterValues(context.Background(), req)

		assert.Error(t, err)
	})
}
