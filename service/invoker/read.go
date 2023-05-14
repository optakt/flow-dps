package invoker

import (
	"fmt"

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

		values, err := index.Values(height, flow.RegisterIDs{regID})
		if err != nil {
			return nil, fmt.Errorf("could not read register: %w", err)
		}
		if len(values) != 1 {
			return nil, fmt.Errorf("wrong number of register values: %d", len(values))
		}

		value := values[0]
		_ = cache.Set(cacheKey, value, int64(len(value)))

		return value, nil
	}
}
