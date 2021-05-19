package feeder_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/service/feeder"
)

func TestFeeder_Delta(t *testing.T) {
	const (
		firstCommit  = "d85b7dc2d6be69c5cc10f0d128595352354e57fbd923ac1ad3f734518610ca73"
		secondCommit = "20a7c8d5447a9acc9cb8de372935669f50645ebd106d98e71a25cf5196595856"
	)

	f, err := feeder.FromLedgerWAL("./testdata")
	require.NoError(t, err)

	// Verify that trying to feed an invalid commit does not work.
	deltas, err := f.Delta(flow.StateCommitment(`invalid_commit`))
	assert.Error(t, err)
	assert.Empty(t, deltas)

	// First state commitment hash after the root hash in test data.
	commit, err := hex.DecodeString(firstCommit)
	require.NoError(t, err)

	// Verify that calling feed on a commit that has deltas and in the right order returns no error and a slice of deltas.
	deltas, err = f.Delta(commit)
	assert.NoError(t, err)
	assert.NotEmpty(t, deltas)

	// Next state commitment hash after the root hash in test data.
	commit, err = hex.DecodeString(secondCommit)
	require.NoError(t, err)

	// Verify that multiple subsequent calls to Feed() work as expected.
	deltas, err = f.Delta(commit)
	assert.NoError(t, err)
	assert.NotEmpty(t, deltas)

	// Trying to feed the first commit again should fail.
	commit, err = hex.DecodeString(firstCommit)
	require.NoError(t, err)

	deltas, err = f.Delta(commit)
	assert.Error(t, err)
	assert.Empty(t, deltas)
}
