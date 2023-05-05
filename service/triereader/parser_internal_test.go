package triereader

import (
	"testing"

	"github.com/onflow/flow-archive/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestFromWAL(t *testing.T) {
	reader := mocks.BaselineWALReader(t)

	feeder := FromWAL(reader)

	assert.Equal(t, reader, feeder.reader)
}
