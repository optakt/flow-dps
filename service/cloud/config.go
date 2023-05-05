package cloud

import (
	"github.com/onflow/flow-go/model/flow"
	"math"
)

const (
	cborMaxArrayElements int = math.MaxInt64 // max capacity allowed by cbor
)

// DefaultConfig is the default configuration for the Google Cloud Streamer.
var DefaultConfig = Config{
	BufferSize:    32,
	CatchupBlocks: []flow.Identifier{},
}

// Config is the configuration for a Google Cloud Streamer.
type Config struct {
	BufferSize    uint
	CatchupBlocks []flow.Identifier
}

// Option is a function that can be applied to a Config.
type Option func(*Config)

// WithBufferSize can be used to specify the buffer size for a
// Google Cloud Streamer to use.
func WithBufferSize(size uint) Option {
	return func(cfg *Config) {
		cfg.BufferSize = size
	}
}

// WithCatchupBlocks injects a number of block IDs that are already finalized,
// but for which we still need to download the execution data records.
func WithCatchupBlocks(blockIDs []flow.Identifier) Option {
	return func(cfg *Config) {
		cfg.CatchupBlocks = blockIDs
	}
}
