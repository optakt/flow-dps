package convert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-archive/models/convert"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestPathsToBytes(t *testing.T) {
	got := convert.PathsToBytes(mocks.GenericLedgerPaths(3))

	assert.Equal(t, [][]byte{
		mocks.ByteSlice(mocks.GenericLedgerPath(0)),
		mocks.ByteSlice(mocks.GenericLedgerPath(1)),
		mocks.ByteSlice(mocks.GenericLedgerPath(2)),
	}, got)
}

func TestBytesToPaths(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		wantPaths := mocks.GenericLedgerPaths(3)

		bb := [][]byte{
			mocks.ByteSlice(mocks.GenericLedgerPath(0)),
			mocks.ByteSlice(mocks.GenericLedgerPath(1)),
			mocks.ByteSlice(mocks.GenericLedgerPath(2)),
		}

		got, err := convert.BytesToPaths(bb)

		assert.NoError(t, err)
		assert.Equal(t, wantPaths, got)
	})

	t.Run("incorrect-length paths should fail", func(t *testing.T) {
		t.Parallel()

		invalidPath := []byte{0x1a, 0x04, 0x57, 0x70, 0x00}

		bb := [][]byte{invalidPath}
		_, err := convert.BytesToPaths(bb)

		assert.Error(t, err)
	})

	t.Run("empty paths should fail", func(t *testing.T) {
		t.Parallel()

		invalidPath := []byte("")

		bb := [][]byte{invalidPath}
		_, err := convert.BytesToPaths(bb)

		assert.Error(t, err)
	})
}
