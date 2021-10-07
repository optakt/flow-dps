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

package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

// Server is the http server that will be serving the /metrics request for prometheus.
type Server struct {
	server *http.Server
	log    zerolog.Logger
}

// NewServer creates a new server that exposes metrics.
func NewServer(log zerolog.Logger, address string) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/debug/pprof/", http.DefaultServeMux)

	m := Server{
		server: &http.Server{
			Addr:    address,
			Handler: mux,
		},
		log: log,
	}

	return &m
}

// Start registers the metrics and launches the server.
func (s *Server) Start() error {
	err := RegisterBadgerMetrics()
	if err != nil {
		return fmt.Errorf("could not register badger metrics: %w", err)
	}

	err = s.server.ListenAndServe()
	if err != nil {
		return fmt.Errorf("could not listen and serve: %w", err)
	}

	return nil
}
