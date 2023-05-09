package archive

import (
	"io"

	"github.com/onflow/flow-go/model/flow"
)

// New pebble-based database interface.
type Library2 interface {
	ReadLibrary2
	WriteLibrary2

	io.Closer
}

type ReadLibrary2 interface {
	GetPayload(height uint64, reg flow.RegisterID) ([]byte, error)
}

type WriteLibrary2 interface {
	BatchSetPayload(height uint64, entries flow.RegisterEntries) error
}
