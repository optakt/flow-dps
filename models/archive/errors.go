package archive

import (
	"errors"
)

// Sentinel errors.
var (
	ErrFinished    = errors.New("finished")
	ErrUnavailable = errors.New("unavailable")
)
