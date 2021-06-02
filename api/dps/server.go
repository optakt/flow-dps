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
	"github.com/optakt/flow-dps/models/dps"
)

// TODO: Add additional layer for API client that makes native Go
// function calls instead of the GRPC structs. We should probably
// redo the state interfaces and implement some of them in both GRPC
// and in native Go on top of Badger.

// TODO: Create a central interface declaration for encoding/decoding that we
// can re-use across components, and add the compression to GRPC.

// Server is a simple implementation of the generated APIServer interface.
// It simply forwards requests to its controller directly without any extra logic.
// It could be used later on to specify GRPC options specifically for certain routes.
type Server struct {
	index dps.IndexReader
	codec cbor.EncMode
}

// NewServer creates a Server given a Controller pointer.
func NewServer(index dps.IndexReader) (*Server, error) {

	codec, _ := cbor.CanonicalEncOptions().EncMode()

	s := Server{
		index: index,
		codec: codec,
	}

	return &s, nil
}

// GetHeader calls the server's controller with the GetHeader method.
func (s *Server) GetHeader(ctx context.Context, req *GetHeaderRequest) (*GetHeaderResponse, error) {

	header, err := s.index.Header(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	data, err := s.codec.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("could not encode header: %w", err)
	}

	res := GetHeaderResponse{
		Height: req.Height,
		Data:   data,
	}

	return &res, nil
}

func (s *Server) GetCommit(ctx context.Context, req *GetCommitRequest) (*GetCommitResponse, error) {

	commit, err := s.index.Commit(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get commit: %w", err)
	}

	res := GetCommitResponse{
		Height: req.Height,
		Commit: commit[:],
	}

	return &res, nil
}

func (s *Server) GetEvents(ctx context.Context, req *GetEventsRequest) (*GetEventsResponse, error) {

	events, err := s.index.Events(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get events: %w", err)
	}

	data, err := s.codec.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("could not encode events: %w", err)
	}

	res := GetEventsResponse{
		Height: req.Height,
		Data:   data,
	}

	return &res, nil
}

// GetRegisters calls the server's controller with the GetRegisters method.
func (s *Server) GetRegisters(ctx context.Context, req *GetRegistersRequest) (*GetRegistersResponse, error) {

	values := make([][]byte, 0, len(req.Paths))
	for _, bytes := range req.Paths {
		path, err := ledger.ToPath(bytes)
		if err != nil {
			return nil, fmt.Errorf("could not convert path (%x): %w", path, err)
		}
		value, err := s.index.Register(req.Height, path)
		if err != nil {
			return nil, fmt.Errorf("could not read register (%x): %w", path, err)
		}
		values = append(values, value)
	}

	res := GetRegistersResponse{
		Height: req.Height,
		Paths:  req.Paths,
		Values: values,
	}

	return &res, nil
}
