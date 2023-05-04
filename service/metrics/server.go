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
