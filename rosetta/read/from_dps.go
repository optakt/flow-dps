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
	"github.com/onflow/flow-go/model/flow"

	grpc "github.com/optakt/flow-dps/api/server"
	"github.com/optakt/flow-dps/rosetta/invoker"
)

func FromDPS(client grpc.APIClient) invoker.ReadFunc {
	return func(commit flow.StateCommitment) delta.GetRegisterFunc {
		readCache := make(map[flow.RegisterID]flow.RegisterValue)
		return func(owner string, controller string, key string) (flow.RegisterValue, error) {

			regID := flow.NewRegisterID(owner, controller, key)
			value, ok := readCache[regID]
			if ok {
				return value, nil
			}

			part1 := grpc.KeyPart{
				Type:  uint64(state.KeyPartOwner),
				Value: []byte(owner),
			}
			part2 := grpc.KeyPart{
				Type:  uint64(state.KeyPartController),
				Value: []byte(controller),
			}
			part3 := grpc.KeyPart{
				Type:  uint64(state.KeyPartKey),
				Value: []byte(key),
			}
			reqKey := grpc.Key{
				Parts: []*grpc.KeyPart{&part1, &part2, &part3},
			}
			req := grpc.GetValuesRequest{
				Hash:    commit[:],
				Version: nil, // use default pathfinder version
				Keys:    []*grpc.Key{&reqKey},
			}

			res, err := client.GetValues(context.Background(), &req)
			if err != nil {
				return nil, fmt.Errorf("could not get get ledger register: %w", err)
			}

			value = flow.RegisterValue(res.Values[0])
			readCache[regID] = value

			return value, nil
		}
	}
}
