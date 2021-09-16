package retriever

import (
	"errors"
)

// Rosetta Sentinel Errors.
var (
	ErrNoAddress    = errors.New("event without address")
	ErrNotSupported = errors.New("unsupported event type")
)
