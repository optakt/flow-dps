package retriever

import (
	"errors"
)

var (
	ErrNoAddress    = errors.New("event without address")
	ErrNotSupported = errors.New("unspported event type")
)
