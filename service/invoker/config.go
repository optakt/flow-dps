package invoker

import (
	"github.com/onflow/flow-go/engine/execution/computation"
	"github.com/onflow/flow-go/fvm/storage/derived"
	"github.com/onflow/flow-go/model/flow"
)

// Config is the configuration for an invoker.
type Config struct {
	computation.ComputationConfig
	CacheSize uint64
	ChainID   flow.ChainID
}

const DefaultCacheSize = uint64(100_000_000) // ~100 MB default size

var DefaultConfig = Config{
	CacheSize: DefaultCacheSize,
	ComputationConfig: computation.ComputationConfig{
		DerivedDataCacheSize: derived.DefaultDerivedDataCacheSize,
	},
}
