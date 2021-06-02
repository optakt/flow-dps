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

package dps

import (
	"context"
	"fmt"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Index struct {
	client APIClient
}

func IndexFromAPI(client APIClient) *Index {

	i := Index{
		client: client,
	}

	return &i
}

func (i *Index) Last() (uint64, error) {

	req := GetLastRequest{}
	res, err := i.client.GetLast(context.Background(), &req)
	if err != nil {
		return 0, fmt.Errorf("could not get last height: %w", err)
	}

	return res.Height, nil
}

func (i *Index) Header(height uint64) (*flow.Header, error) {

	req := GetHeaderRequest{
		Height: height,
	}
	res, err := i.client.GetHeader(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	var header flow.Header
	err = cbor.Unmarshal(res.Data, &header)
	if err != nil {
		return nil, fmt.Errorf("could not decode header: %w", err)
	}

	return &header, nil
}

func (i *Index) Commit(height uint64) (flow.StateCommitment, error) {

	req := GetCommitRequest{
		Height: height,
	}
	res, err := i.client.GetCommit(context.Background(), &req)
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not get commit: %w", err)
	}

	commit, err := flow.ToStateCommitment(res.Commit)
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not convert commit: %w", err)
	}

	return commit, nil
}

func (i *Index) Events(height uint64, types ...flow.EventType) ([]flow.Event, error) {

	tt := make([]string, 0, len(types))
	for _, typ := range types {
		tt = append(tt, string(typ))
	}

	req := GetEventsRequest{
		Height: height,
		Types:  tt,
	}
	res, err := i.client.GetEvents(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get events: %w", err)
	}

	var events []flow.Event
	err = cbor.Unmarshal(res.Data, &events)
	if err != nil {
		return nil, fmt.Errorf("could not decode events: %w", err)
	}

	return events, nil
}

// TODO: Find a way to batch up register requests for Cadence execution so we
// don't have to request them one by one over GRPC.

func (i *Index) Registers(height uint64, paths []ledger.Path) ([]ledger.Value, error) {

	pp := make([][]byte, 0, len(paths))
	for _, path := range paths {
		pp = append(pp, path[:])
	}

	req := GetRegistersRequest{
		Height: height,
		Paths:  pp,
	}
	res, err := i.client.GetRegisters(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get registers: %w", err)
	}

	values := make([]ledger.Value, 0, len(res.Values))
	for _, value := range res.Values {
		values = append(values, ledger.Value(value))
	}

	return values, nil
}
