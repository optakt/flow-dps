package retriever

import (
	"errors"
)

// Rosetta Sentinel Errors.
var (
	ErrNoAddress    = errors.New("event without address")
	ErrNotSupported = errors.New("unsupported event type")
)

const (
	// Cadence error returned when it was not possible to borrow the vault reference.
	// This can happen if the account does not exist at the given height.
	missingVault = "Could not borrow Balance reference to the Vault"

	// Error description for failure to find a transaction.
	txMissing = "transaction not found in given block"
)
