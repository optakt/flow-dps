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

package tracker

import (
	"testing"

	"github.com/gammazero/deque"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewExecution(t *testing.T) {
	header := mocks.GenericHeader
	blockID := header.ID()
	seal := mocks.GenericSeal(0)

	t.Run("nominal case", func(t *testing.T) {
		log := zerolog.Nop()
		stream := mocks.BaselineRecordStreamer(t)

		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.InsertRootHeight(header.Height)))
		require.NoError(t, db.Update(operation.IndexBlockHeight(header.Height, blockID)))
		require.NoError(t, db.Update(operation.InsertHeader(blockID, header)))
		require.NoError(t, db.Update(operation.IndexBlockSeal(blockID, seal.ID())))
		require.NoError(t, db.Update(operation.InsertSeal(seal.ID(), seal)))

		exec, err := NewExecution(log, db, stream)

		require.NoError(t, err)
		assert.Equal(t, stream, exec.stream)
		assert.NotNil(t, exec.queue)
		assert.NotNil(t, exec.records)
		assert.NotEmpty(t, exec.records)
	})

	t.Run("handles missing root height", func(t *testing.T) {
		log := zerolog.Nop()
		stream := mocks.BaselineRecordStreamer(t)

		db := helpers.InMemoryDB(t)
		// Do not insert root height.
		//require.NoError(t, db.Update(operation.InsertRootHeight(header.Height)))
		require.NoError(t, db.Update(operation.IndexBlockHeight(header.Height, blockID)))
		require.NoError(t, db.Update(operation.InsertHeader(blockID, header)))
		require.NoError(t, db.Update(operation.IndexBlockSeal(blockID, seal.ID())))
		require.NoError(t, db.Update(operation.InsertSeal(seal.ID(), seal)))

		_, err := NewExecution(log, db, stream)

		assert.Error(t, err)
	})

	t.Run("handles missing block height", func(t *testing.T) {
		log := zerolog.Nop()
		stream := mocks.BaselineRecordStreamer(t)

		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.InsertRootHeight(header.Height)))
		// Do not insert block height.
		//require.NoError(t, db.Update(operation.IndexBlockHeight(header.Height, blockID)))
		require.NoError(t, db.Update(operation.InsertHeader(blockID, header)))
		require.NoError(t, db.Update(operation.IndexBlockSeal(blockID, seal.ID())))
		require.NoError(t, db.Update(operation.InsertSeal(seal.ID(), seal)))

		_, err := NewExecution(log, db, stream)

		assert.Error(t, err)
	})

	t.Run("handles missing header", func(t *testing.T) {
		log := zerolog.Nop()
		stream := mocks.BaselineRecordStreamer(t)

		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.InsertRootHeight(header.Height)))
		require.NoError(t, db.Update(operation.IndexBlockHeight(header.Height, blockID)))
		// Do not insert header.
		//require.NoError(t, db.Update(operation.InsertHeader(blockID, header)))
		require.NoError(t, db.Update(operation.IndexBlockSeal(blockID, seal.ID())))
		require.NoError(t, db.Update(operation.InsertSeal(seal.ID(), seal)))

		_, err := NewExecution(log, db, stream)

		assert.Error(t, err)
	})

	t.Run("handles missing seal index", func(t *testing.T) {
		log := zerolog.Nop()
		stream := mocks.BaselineRecordStreamer(t)

		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.InsertRootHeight(header.Height)))
		require.NoError(t, db.Update(operation.IndexBlockHeight(header.Height, blockID)))
		require.NoError(t, db.Update(operation.InsertHeader(blockID, header)))
		// Do not insert seal ID.
		//require.NoError(t, db.Update(operation.IndexBlockSeal(blockID, seal.ID())))
		require.NoError(t, db.Update(operation.InsertSeal(seal.ID(), seal)))

		_, err := NewExecution(log, db, stream)

		assert.Error(t, err)
	})

	t.Run("handles missing seal", func(t *testing.T) {
		log := zerolog.Nop()
		stream := mocks.BaselineRecordStreamer(t)

		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.InsertRootHeight(header.Height)))
		require.NoError(t, db.Update(operation.IndexBlockHeight(header.Height, blockID)))
		require.NoError(t, db.Update(operation.InsertHeader(blockID, header)))
		require.NoError(t, db.Update(operation.IndexBlockSeal(blockID, seal.ID())))
		// Do not insert seal.
		//require.NoError(t, db.Update(operation.InsertSeal(seal.ID(), seal)))

		_, err := NewExecution(log, db, stream)

		assert.Error(t, err)
	})
}

func TestExecution_Purge(t *testing.T) {
	blockIDs := mocks.GenericBlockIDs(4)
	var blocks []*uploader.BlockData
	for idx := range blockIDs {
		blocks = append(blocks, &uploader.BlockData{
			Block: &flow.Block{
				Header: &flow.Header{
					Height: uint64(idx), // Block height is equal to its index in the slice.
				},
			},
		})
	}

	tests := []struct {
		name string

		threshold uint64
		before    map[flow.Identifier]*uploader.BlockData

		after map[flow.Identifier]*uploader.BlockData
	}{
		{
			name: "nominal case",

			threshold: 2,
			before: map[flow.Identifier]*uploader.BlockData{
				blockIDs[0]: blocks[0],
				blockIDs[1]: blocks[1],
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},
		},
		{
			name: "purge threshold vastly over max height",

			threshold: 42,
			before: map[flow.Identifier]*uploader.BlockData{
				blockIDs[0]: blocks[0],
				blockIDs[1]: blocks[1],
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{},
		},
		{
			name: "purge threshold at 0",

			threshold: 0,
			before: map[flow.Identifier]*uploader.BlockData{
				blockIDs[0]: blocks[0],
				blockIDs[1]: blocks[1],
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{
				blockIDs[0]: blocks[0],
				blockIDs[1]: blocks[1],
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},
		},
		{
			name: "purge threshold at 1",

			threshold: 1,
			before: map[flow.Identifier]*uploader.BlockData{
				blockIDs[0]: blocks[0],
				blockIDs[1]: blocks[1],
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{
				blockIDs[1]: blocks[1],
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},
		},
		{
			name: "purge threshold just below last record height",

			threshold: 3,
			before: map[flow.Identifier]*uploader.BlockData{
				blockIDs[0]: blocks[0],
				blockIDs[1]: blocks[1],
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{
				blockIDs[3]: blocks[3],
			},
		},
		{
			name: "purge threshold exactly at last record height",

			threshold: 4,
			before: map[flow.Identifier]*uploader.BlockData{
				blockIDs[0]: blocks[0],
				blockIDs[1]: blocks[1],
				blockIDs[2]: blocks[2],
				blockIDs[3]: blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			exec := BaselineExecution(t)
			exec.records = test.before

			exec.purge(test.threshold)

			assert.Len(t, exec.records, len(test.after))
			assert.Equal(t, test.after, exec.records)
		})
	}

}

func BaselineExecution(t *testing.T, opts ...func(*Execution)) *Execution {
	t.Helper()

	e := Execution{
		log:     zerolog.Nop(),
		queue:   deque.New(),
		stream:  mocks.BaselineRecordStreamer(t),
		records: make(map[flow.Identifier]*uploader.BlockData),
	}

	for _, opt := range opts {
		opt(&e)
	}

	return &e
}

func WithLogger(log zerolog.Logger) func(*Execution) {
	return func(execution *Execution) {
		execution.log = log
	}
}

func WithStreamer(stream RecordStreamer) func(*Execution) {
	return func(execution *Execution) {
		execution.stream = stream
	}
}

func WithQueue(queue *deque.Deque) func(*Execution) {
	return func(execution *Execution) {
		execution.queue = queue
	}
}
