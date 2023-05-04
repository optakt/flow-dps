package tracker_test

import (
	"testing"

	"github.com/gammazero/deque"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/engine/execution/ingestion/uploader"

	"github.com/onflow/flow-archive/service/tracker"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestExecution_Update(t *testing.T) {
	record := mocks.GenericRecord()
	update := record.TrieUpdates

	t.Run("nominal case with nothing in queue", func(t *testing.T) {
		t.Parallel()

		streamer := mocks.BaselineRecordStreamer(t)
		streamer.NextFunc = func() (*uploader.BlockData, error) {
			return record, nil
		}

		exec := tracker.BaselineExecution(t, tracker.WithStreamer(streamer))

		got, err := exec.AllUpdates()

		require.NoError(t, err)
		assert.Equal(t, record.TrieUpdates, got)
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

		got, err := exec.AllUpdates()

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

		_, err := exec.AllUpdates()

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
		_, err := exec.AllUpdates()

		assert.NoError(t, err)

		// The next call loads the same block, realizes something is wrong and returns an error.
		_, err = exec.AllUpdates()

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
