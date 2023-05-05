package invoker

import (
	"fmt"

	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-archive/models/archive"
)

func readRegister(
	index archive.Reader,
	cache Cache,
	height uint64,
) func(flow.RegisterID) (flow.RegisterValue, error) {
	return func(regID flow.RegisterID) (flow.RegisterValue, error) {
		cacheKey := fmt.Sprintf("%d/%s", height, regID)
		cacheValue, ok := cache.Get(cacheKey)
		if ok {
			return cacheValue.(flow.RegisterValue), nil
		}

		path, err := pathfinder.KeyToPath(
			state.RegisterIDToKey(regID),
			complete.DefaultPathFinderVersion)
		if err != nil {
			return nil, fmt.Errorf("could not convert key to path: %w", err)
		}

		values, err := index.Values(height, []ledger.Path{path})
		if err != nil {
			return nil, fmt.Errorf("could not read register: %w", err)
		}

		value := flow.RegisterValue(values[0])
		_ = cache.Set(cacheKey, value, int64(len(value)))

		return value, nil
	}
}
