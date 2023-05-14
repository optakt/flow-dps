package convert_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-archive/models/convert"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestTypesToStrings(t *testing.T) {
	types := mocks.GenericEventTypes(4)

	got := convert.TypesToStrings(types)

	for _, typ := range types {
		assert.Contains(t, got, string(typ))
	}
}

func TestStringsToTypes(t *testing.T) {
	types := mocks.GenericEventTypes(4)

	var ss []string
	for _, typ := range types {
		ss = append(ss, string(typ))
	}

	got := convert.StringsToTypes(ss)

	assert.Equal(t, types, got)
}

func TestRosettaTime(t *testing.T) {
	ti := time.Now()

	got := convert.RosettaTime(ti)

	assert.Equal(t, ti.UnixNano()/1_000_000, got)
}
