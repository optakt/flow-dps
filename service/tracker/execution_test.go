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

package tracker_test

import (
	"testing"

	"github.com/gammazero/deque"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"

	"github.com/onflow/flow-dps/service/tracker"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestExecution_Update(t *testing.T) {
	record := mocks.GenericRecord()
	update := mocks.GenericTrieUpdate(5)

	t.Run("nominal case with nothing in queue", func(t *testing.T) {
		t.Parallel()

		streamer := mocks.BaselineRecordStreamer(t)
		streamer.NextFunc = func() (*uploader.BlockData, error) {
			return record, nil
		}

		exec := tracker.BaselineExecution(t, tracker.WithStreamer(streamer))

		got, err := exec.Update()

		require.NoError(t, err)
		assert.Equal(t, record.TrieUpdates[0], got)
	})

	t.Run("nominal case with queue already filled", func(t *testing.T) {
		t.Parallel()

		streamer := mocks.BaselineRecordStreamer(t)
		streamer.NextFunc = func() (*uploader.BlockData, error) {
			t.Fatal("unexpected call to streamer.Next()") // This should never be called, since the queue is already filled.
			return record, nil
		}

		queue := deque.New()
		queue.PushBack(update)

		exec := tracker.BaselineExecution(
			t,
			tracker.WithQueue(queue),
			tracker.WithStreamer(streamer),
		)

		got, err := exec.Update()

		require.NoError(t, err)
		assert.Equal(t, update, got)
	})

	t.Run("handles streamer failure on Next", func(t *testing.T) {
		t.Parallel()

		streamer := mocks.BaselineRecordStreamer(t)
		streamer.NextFunc = func() (*uploader.BlockData, error) {
			return nil, mocks.GenericError
		}

		exec := tracker.BaselineExecution(t, tracker.WithStreamer(streamer))

		_, err := exec.Update()

		assert.Error(t, err)
	})

	t.Run("handles duplicate records", func(t *testing.T) {
		t.Parallel()

		// Only keep one trie update per block to make each update call go through one block.
		smallBlock := mocks.GenericRecord()
		smallBlock.TrieUpdates = smallBlock.TrieUpdates[:1]

		streamer := mocks.BaselineRecordStreamer(t)
		streamer.NextFunc = func() (*uploader.BlockData, error) {
			return smallBlock, nil
		}

		exec := tracker.BaselineExecution(t, tracker.WithStreamer(streamer))

		// The first call loads our "small block" with only one trie update and consumes it.
		_, err := exec.Update()

		assert.NoError(t, err)

		// The next call loads the same block, realizes something is wrong and returns an error.
		_, err = exec.Update()

		assert.Error(t, err)
	})
}

func TestExecution_Record(t *testing.T) {
	record := mocks.GenericRecord()

	t.Run("nominal case with nothing in queue", func(t *testing.T) {
		t.Parallel()

		streamer := mocks.BaselineRecordStreamer(t)
		streamer.NextFunc = func() (*uploader.BlockData, error) {
			return record, nil
		}

		exec := tracker.BaselineExecution(t, tracker.WithStreamer(streamer))

		got, err := exec.Record(record.Block.ID())

		require.NoError(t, err)
		assert.Equal(t, record, got)
	})

	t.Run("handles streamer failure on Next", func(t *testing.T) {
		t.Parallel()

		streamer := mocks.BaselineRecordStreamer(t)
		streamer.NextFunc = func() (*uploader.BlockData, error) {
			return nil, mocks.GenericError
		}

		exec := tracker.BaselineExecution(t, tracker.WithStreamer(streamer))

		_, err := exec.Record(record.Block.ID())

		assert.Error(t, err)
	})
}
