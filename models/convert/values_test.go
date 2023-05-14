package convert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-archive/models/convert"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestValuesToBytes(t *testing.T) {
	values := mocks.GenericRegisterValues(4)

	var bb [][]byte
	for _, val := range values {
		bb = append(bb, val[:])
	}

	got := convert.ValuesToBytes(values)

	assert.Equal(t, bb, got)
}

func TestBytesToValues(t *testing.T) {
	values := mocks.GenericRegisterValues(4)

	var bb [][]byte
	for _, val := range values {
		bb = append(bb, val[:])
	}

	got := convert.BytesToValues(bb)

	assert.Equal(t, values, got)
}
