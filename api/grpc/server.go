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

import "context"

// Server is a simple implementation of the generated APIServer interface.
// It simply forwards requests to its controller directly without any extra logic.
// It could be used later on to specify GRPC options specifically for certain routes.
type Server struct {
	ctrl *Controller
}

// NewServer creates a Server given a Controller pointer.
func NewServer(ctrl *Controller) *Server {
	return &Server{
		ctrl: ctrl,
	}
}

// GetRegister calls the server's controller with the GetRegister method.
func (s *Server) GetRegister(ctx context.Context, req *GetRegisterRequest) (*GetRegisterResponse, error) {
	return s.ctrl.GetRegister(ctx, req)
}

// GetValues calls the server's controller with the GetValues method.
func (s *Server) GetValues(ctx context.Context, req *GetValuesRequest) (*GetValuesResponse, error) {
	return s.ctrl.GetValues(ctx, req)
}
