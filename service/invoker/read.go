// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package invoker

import (
	"fmt"

	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-dps/models/dps"
)

func readRegister(index dps.Reader, cache Cache, height uint64) delta.GetRegisterFunc {
	return func(owner string, controller string, key string) (flow.RegisterValue, error) {

		cacheKey := fmt.Sprintf("%d/%x/%x/%s", height, owner, controller, key)
		cacheValue, ok := cache.Get(cacheKey)
		if ok {
			return cacheValue.(flow.RegisterValue), nil
		}

		regID := flow.NewRegisterID(owner, controller, key)
		path, err := pathfinder.KeyToPath(state.RegisterIDToKey(regID), complete.DefaultPathFinderVersion)
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
