package convert

import (
	"github.com/onflow/flow-go/model/flow"
)

// IDToHash converts a flow Identifier into a byte slice.
func IDToHash(id flow.Identifier) []byte {
	hash := make([]byte, 32)
	copy(hash, id[:])

	return hash
}

// CommitToHash converts a flow StateCommitment into a byte slice.
func CommitToHash(commit flow.StateCommitment) []byte {
	hash := make([]byte, 32)
	copy(hash, commit[:])

	return hash
}
