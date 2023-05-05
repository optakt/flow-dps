package convert

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
)

// PathsToBytes converts a slice of ledger paths into a slice of byte slices.
func PathsToBytes(paths []ledger.Path) [][]byte {
	bb := make([][]byte, 0, len(paths))
	for _, path := range paths {
		b := make([]byte, len(path))
		copy(b, path[:])
		bb = append(bb, b)
	}
	return bb
}

// BytesToPaths converts a slice of byte slices into a slice of ledger paths.
func BytesToPaths(bb [][]byte) ([]ledger.Path, error) {
	paths := make([]ledger.Path, 0, len(bb))
	for _, b := range bb {
		path, err := ledger.ToPath(b)
		if err != nil {
			return nil, fmt.Errorf("could not convert path (%x): %w", b, err)
		}
		paths = append(paths, path)
	}
	return paths, nil
}
