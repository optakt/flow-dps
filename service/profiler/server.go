// Copyright 2022 Dapper Labs, Inc.
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

package profiler

import (
	"fmt"
	"net/http"

	_ "net/http/pprof"

	"github.com/rs/zerolog"
)

// Server is the http server that will be serving the /debug/pprof/ request for pprof.
type Server struct {
	server *http.Server
	log    zerolog.Logger
}

// NewServer creates a new server that exposes pprof endpoint.
func NewServer(log zerolog.Logger, address string) *Server {
	m := Server{
		server: &http.Server{
			Addr:    address,
			Handler: http.DefaultServeMux,
		},
		log: log,
	}

	return &m
}

// Start registers the pprof and launches the server.
func (s *Server) Start() error {
	err := s.server.ListenAndServe()
	if err != nil {
		return fmt.Errorf("could not listen and serve: %w", err)
	}

	return nil
}
