package mapper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithSkipRegisters(t *testing.T) {
	c := Config{
		SkipRegisters: false,
	}
	skip := true

	WithSkipRegisters(skip)(&c)

	assert.Equal(t, skip, c.SkipRegisters)
}

func TestWithIndexHeader(t *testing.T) {
	c := &Config{
		WaitInterval: time.Second,
	}
	interval := time.Millisecond

	WithWaitInterval(interval)(c)

	assert.Equal(t, interval, c.WaitInterval)
}
