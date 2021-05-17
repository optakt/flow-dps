package feeder_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/awfm9/flow-dps/service/feeder"
)

func TestFeeder_Delta(t *testing.T) {
	f, err := feeder.FromLedgerWAL("./testdata")
	require.NoError(t, err)

	deltas1, err := f.Delta([]byte{})
	assert.NoError(t, err)
	assert.NotEmpty(t, deltas1)

	// Verify that multiple subsequent calls to Delta() work as expected.
	deltas2, err := f.Delta([]byte{})
	assert.NoError(t, err)
	assert.NotEmpty(t, deltas2)

	// Verify that both calls returned different deltas.
	assert.NotEqual(t, deltas1, deltas2)
}
