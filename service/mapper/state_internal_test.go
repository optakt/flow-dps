package mapper

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyState(t *testing.T) {
	s := EmptyState("root.checkpoint")

	assert.Equal(t, StatusInitialize, s.status)
	assert.Equal(t, s.height, uint64(math.MaxUint64))
	assert.NotNil(t, s.registers)
	assert.Empty(t, s.registers)
	assert.NotNil(t, s.updates)
	assert.Empty(t, s.updates)
	assert.NotNil(t, s.done)
}
