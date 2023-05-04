package storage

import (
	"github.com/onflow/flow-archive/models/archive"
)

// Library is the storage library.
type Library struct {
	codec archive.Codec
}

// New returns a new storage library using the given codec.
func New(codec archive.Codec) *Library {
	lib := Library{
		codec: codec,
	}

	return &lib
}
