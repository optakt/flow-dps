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
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/optakt/flow-dps/api/server"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/state"
)

func main() {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagLog   string
		flagIndex string
		flagPort  uint16
	)

	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLog, "log", "l", "info", "log output level")
	pflag.Uint16VarP(&flagPort, "port", "p", 5005, "port to serve GRPC API on")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLog)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// Initialize the index core state.
	index, err := badger.Open(dps.DefaultOptions(flagIndex).WithReadOnly(true))
	if err != nil {
		log.Fatal().Err(err).Msg("could not open index DB")
	}
	core, err := state.NewCore(index)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize ledger")
	}

	// GRPC API initialization.
	controller := server.NewController(core)
	svr := grpc.NewServer()

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	go func() {
		log.Info().Msg("starting GRPC API server")
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", flagPort))
		if err != nil {
			log.Fatal().Err(err).Uint16("port", flagPort).Msg("could not listen")
		}
		server.RegisterAPIServer(svr, server.New(controller))
		err = svr.Serve(listener)
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
		svr.GracefulStop()
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
