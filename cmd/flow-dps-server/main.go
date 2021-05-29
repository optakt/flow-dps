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

package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	gsvr "google.golang.org/grpc"

	"github.com/optakt/flow-dps/api/grpc"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/state"
)

func main() {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagLevel    string
		flagIndex    string
		flagHostGRPC string
	)

	pflag.StringVarP(&flagLevel, "log-level", "l", "info", "log output level")
	pflag.StringVarP(&flagIndex, "index-dir", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagHostGRPC, "grpc-host", "g", ":5005", "host URL for GRPC API endpoint")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// Initialize the index core state.
	index, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Fatal().Err(err).Msg("could not open index DB")
	}
	core, err := state.NewCore(index)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize ledger")
	}

	// GRPC API initialization.
	gctrl := grpc.NewController(core)
	gsvr := gsvr.NewServer()

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	go func() {
		log.Info().Msg("starting GRPC API server")
		lis, err := net.Listen("tcp", flagHostGRPC)
		if err != nil {
			log.Fatal().Err(err).Str("host", flagHostGRPC).Msg("could not listen")
		}
		grpc.RegisterAPIServer(gsvr, grpc.NewServer(gctrl))
		err = gsvr.Serve(lis)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("GRPC API encountered error")
		}
		log.Info().Msg("GRPC API server stopped")
	}()

	<-sig

	log.Info().Msg("startup complete")

	// The following code starts a shut down with a certain timeout and makes
	// sure that the main executing components are shutting down within the
	// allocated shutdown time. Otherwise, we will force the shutdown and log
	// an error. We then wait for shutdown on each component to complete.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		log.Info().Msg("shutting down GRPC API")
		defer wg.Done()
		gsvr.GracefulStop()
		log.Info().Msg("GRPC API shutdown complete")
	}()
	go func() {
		<-sig
		log.Info().Msg("forcing exit")
		os.Exit(1)
	}()

	wg.Wait()

	os.Exit(0)
}
