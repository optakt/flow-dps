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
	"context"
	"fmt"

	"github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/rosetta/invoker"
)

func FromDPS(client dps.APIClient) invoker.ReadFunc {
	heightCache := make(map[flow.StateCommitment]uint64)
	return func(commit flow.StateCommitment) delta.GetRegisterFunc {
		readCache := make(map[flow.RegisterID]flow.RegisterValue)
		return func(owner string, controller string, key string) (flow.RegisterValue, error) {

			// If we have already cached the register at this commit, return the
			// value immediately.
			regID := flow.NewRegisterID(owner, controller, key)
			value, ok := readCache[regID]
			if ok {
				return value, nil
			}

			// Next, we check if we already have a height in our height cache so
			// we can avoid doing an extra request over the network.
			height, ok := heightCache[commit]
			if !ok {
				_ = ok
				// FIXME: have a way to get the height from the commit
			}

			path, err := pathfinder.KeyToPath(state.RegisterIDToKey(regID), complete.DefaultPathFinderVersion)
			if err != nil {
				return nil, fmt.Errorf("could not convert key to path: %w", err)
			}

			// TODO: Add additional layer for API client that makes native Go
			// function calls instead of the GRPC structs. We should probably
			// redo the state interfaces and implement some of them in both GRPC
			// and in native Go on top of Badger.

			req := dps.ReadRegistersRequest{
				Height: &height,
				Paths:  [][]byte{path[:]},
			}
			res, err := client.ReadRegisters(context.Background(), &req)
			if err != nil {
				return nil, fmt.Errorf("could not get get ledger register: %w", err)
			}

			value = flow.RegisterValue(res.Values[0])
			readCache[regID] = value

			return value, nil
		}
	}
}
