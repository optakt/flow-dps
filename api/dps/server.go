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
)

// TODO: Add additional layer for API client that makes native Go
// function calls instead of the GRPC structs. We should probably
// redo the state interfaces and implement some of them in both GRPC
// and in native Go on top of Badger.

// Server is a simple implementation of the generated APIServer interface.
// It simply forwards requests to its controller directly without any extra logic.
// It could be used later on to specify GRPC options specifically for certain routes.
type Server struct {
	ctrl  *Controller
	codec cbor.EncMode
}

// NewServer creates a Server given a Controller pointer.
func NewServer(ctrl *Controller) (*Server, error) {

	codec, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, fmt.Errorf("could not initialize encoder: %w", err)
	}

	s := Server{
		ctrl:  ctrl,
		codec: codec,
	}

	return &s, nil
}

// GetHeader calls the server's controller with the GetHeader method.
func (s *Server) GetHeader(ctx context.Context, req *GetHeaderRequest) (*GetHeaderResponse, error) {

	header, height, err := s.ctrl.GetHeader(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	data, err := s.codec.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("could not encode header: %w", err)
	}

	res := GetHeaderResponse{
		Height: height,
		Data:   data,
	}

	return &res, nil
}

// ReadRegisters calls the server's controller with the ReadRegisters method.
func (s *Server) ReadRegisters(ctx context.Context, req *ReadRegistersRequest) (*ReadRegistersResponse, error) {

	paths, err := toPaths(req.Paths)
	if err != nil {
		return nil, fmt.Errorf("could not convert paths: %w", err)
	}

	values, height, err := s.ctrl.ReadRegisters(req.Height, paths)
	if err != nil {
		return nil, fmt.Errorf("could not read registers: %w", err)
	}

	res := ReadRegistersResponse{
		Height: height,
		Paths:  req.Paths,
		Values: toBytes(values),
	}

	return &res, nil
}

func toPaths(bb [][]byte) ([]ledger.Path, error) {
	paths := make([]ledger.Path, 0, len(bb))
	for _, b := range bb {
		path, err := ledger.ToPath(b)
		if err != nil {
			return nil, fmt.Errorf("could not convert path (%x): %w", b, err)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func toBytes(values []ledger.Value) [][]byte {
	bb := make([][]byte, 0, len(values))
	for _, value := range values {
		bb = append(bb, value[:])
	}
	return bb
}
