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

	"github.com/onflow/flow-dps/testing/helpers"
	"github.com/onflow/flow-dps/testing/mocks"
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
	blocks := []*uploader.BlockData{
		{Block: &flow.Block{Header: &flow.Header{Height: 4}}},
		{Block: &flow.Block{Header: &flow.Header{Height: 5}}},
		{Block: &flow.Block{Header: &flow.Header{Height: 6}}},
		{Block: &flow.Block{Header: &flow.Header{Height: 7}}},
	}

	tests := []struct {
		name string

		threshold uint64
		before    map[flow.Identifier]*uploader.BlockData

		after map[flow.Identifier]*uploader.BlockData
	}{
		{
			name: "threshold is at lowest height",

			threshold: blocks[0].Block.Header.Height,
			before: map[flow.Identifier]*uploader.BlockData{
				blocks[0].Block.ID(): blocks[0],
				blocks[1].Block.ID(): blocks[1],
				blocks[2].Block.ID(): blocks[2],
				blocks[3].Block.ID(): blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{
				blocks[0].Block.ID(): blocks[0],
				blocks[1].Block.ID(): blocks[1],
				blocks[2].Block.ID(): blocks[2],
				blocks[3].Block.ID(): blocks[3],
			},
		},
		{
			name: "threshold is above highest height",

			threshold: blocks[3].Block.Header.Height + 1,
			before: map[flow.Identifier]*uploader.BlockData{
				blocks[0].Block.ID(): blocks[0],
				blocks[1].Block.ID(): blocks[1],
				blocks[2].Block.ID(): blocks[2],
				blocks[3].Block.ID(): blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{},
		},
		{
			name: "threshold is in-between",

			threshold: blocks[2].Block.Header.Height,
			before: map[flow.Identifier]*uploader.BlockData{
				blocks[0].Block.ID(): blocks[0],
				blocks[1].Block.ID(): blocks[1],
				blocks[2].Block.ID(): blocks[2],
				blocks[3].Block.ID(): blocks[3],
			},

			after: map[flow.Identifier]*uploader.BlockData{
				blocks[2].Block.ID(): blocks[2],
				blocks[3].Block.ID(): blocks[3],
			},
		},
		{
			name: "does nothing when there is nothing to purge",

			threshold: blocks[2].Block.Header.Height,
			before:    map[flow.Identifier]*uploader.BlockData{},

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
