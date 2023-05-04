package convert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-archive/models/convert"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestIDToHash(t *testing.T) {
	blockID := mocks.GenericHeader.ID()
	got := convert.IDToHash(blockID)
	assert.Equal(t, blockID[:], got)
}

func TestCommitToHash(t *testing.T) {
	got := convert.CommitToHash(mocks.GenericCommit(0))
	assert.Equal(t, mocks.ByteSlice(mocks.GenericCommit(0)), got)
}
