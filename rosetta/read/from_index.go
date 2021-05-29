// Copyright 2021 Alvalor S.A.
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

package read

import (
	"fmt"

	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/invoker"
)

func FromIndex(core dps.State) invoker.ReadFunc {
	return func(commit flow.StateCommitment) delta.GetRegisterFunc {
		readCache := make(map[flow.RegisterID]flow.RegisterValue)
		return func(owner string, controller string, key string) (flow.RegisterValue, error) {

			regID := flow.NewRegisterID(owner, controller, key)
			value, ok := readCache[regID]
			if ok {
				return value, nil
			}

			lkey := state.RegisterIDToKey(regID)
			query, err := ledger.NewQuery(ledger.State(commit), []ledger.Key{lkey})
			if err != nil {
				return nil, fmt.Errorf("could not create ledger query: %w", err)
			}

			values, err := core.Ledger().Get(query)
			if err != nil {
				fmt.Println(err)
				return nil, fmt.Errorf("could not get ledger register: %w", err)
			}
			if len(values) == 0 {
				return nil, nil
			}

			value = flow.RegisterValue(values[0])
			readCache[regID] = value

			return values[0], nil
		}
	}
}
