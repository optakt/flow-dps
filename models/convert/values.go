package convert

import (
	"github.com/onflow/flow-go/ledger"
)

// ValuesToBytes converts a slice of ledger values into a slice of byte slices.
func ValuesToBytes(values []ledger.Value) [][]byte {
	bb := make([][]byte, 0, len(values))
	for _, value := range values {
		b := make([]byte, len(value))
		copy(b, value[:])
		bb = append(bb, b)
	}
	return bb
}

// BytesToValues converts a slice of byte slices into a slice of ledger values.
func BytesToValues(bb [][]byte) []ledger.Value {
	values := make([]ledger.Value, 0, len(bb))
	for _, b := range bb {
		value := ledger.Value(b)
		values = append(values, value)
	}
	return values
}
