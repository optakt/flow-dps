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

package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/onflow/flow-go/ledger"

	"github.com/optakt/flow-dps/models/dps"
)

type Controller struct {
	state dps.State
}

// NewController creates a controller using the given state store.
func NewController(state dps.State) (*Controller, error) {
	c := &Controller{
		state: state,
	}
	return c, nil
}

// GetRegister gets a single register from a given key. Block height can also be given as an optional parameter.
// If no height is given, the last height present within the state is used.
func (c *Controller) GetRegister(_ context.Context, req *GetRegisterRequest, _ ...grpc.CallOption) (*GetRegisterResponse, error) {
	state := c.state.Raw()

	height := c.state.Last().Height()
	if req.Height != nil {
		height = *req.Height
	}

	state = state.WithHeight(height)

	value, err := state.Get(req.Key)
	if err != nil {
		return nil, fmt.Errorf("could not get register in GRPC API: %w", err)
	}

	res := GetRegisterResponse{
		Height: height,
		Key:    req.Key,
		Value:  value,
	}

	return &res, nil
}

// GetValues returns the payload value of an encoded Ledger entry in the same way
// as the Flow Ledger interface would. It takes an input that emulates the `ledger.Query` struct.
// The state hash and the pathfinder key version are optional as part of the request.
// If omitted, the state hash of the latest sealed block and the default pathfinder key encoding is used.
func (c *Controller) GetValues(_ context.Context, req *GetValuesRequest, _ ...grpc.CallOption) (*GetValuesResponse, error) {
	state := c.state.Ledger()

	if req.Version != nil {
		state = state.WithVersion(uint8(*req.Version))
	}

	commit := c.state.Last().Commit()
	if req.Hash != nil {
		commit = req.Hash
	}

	var keys []ledger.Key
	for _, key := range req.Keys {
		var k ledger.Key
		for _, part := range key.Parts {
			k.KeyParts = append(k.KeyParts, ledger.NewKeyPart(uint16(part.Type), part.Value))
		}
		keys = append(keys, k)
	}

	query, err := ledger.NewQuery(commit, keys)
	if err != nil {
		return nil, fmt.Errorf("could not forge query in GRPC API: %w", err)
	}

	values, err := state.Get(query)
	if err != nil {
		return nil, fmt.Errorf("could not get values in GRPC API: %w", err)
	}

	// Convert the ledger.Values into [][]byte.
	var vv [][]byte
	for _, value := range values {
		vv = append(vv, value)
	}

	res := GetValuesResponse{
		Values: vv,
	}

	return &res, nil
}
