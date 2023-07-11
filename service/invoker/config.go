package invoker

import (
	"github.com/onflow/flow-go/engine/execution/computation"
	"github.com/onflow/flow-go/engine/execution/computation/query"
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

// archiveExecutionTimeMultiplier is used to multiply the default time limits for script
// execution. This is because archive node execute scripts slower than execution node,
// but also because archive node can execute scripts longer because archive nodes are
// easier to scale.
const archiveExecutionTimeMultiplier = 10

var DefaultConfig = Config{
	CacheSize: DefaultCacheSize,
	ComputationConfig: computation.ComputationConfig{
		QueryConfig: query.QueryConfig{
			LogTimeThreshold:    query.DefaultLogTimeThreshold * archiveExecutionTimeMultiplier,
			ExecutionTimeLimit:  query.DefaultExecutionTimeLimit * archiveExecutionTimeMultiplier,
			MaxErrorMessageSize: query.DefaultMaxErrorMessageSize,
		},
		DerivedDataCacheSize: derived.DefaultDerivedDataCacheSize,
	},
	ChainID: flow.Emulator,
}
