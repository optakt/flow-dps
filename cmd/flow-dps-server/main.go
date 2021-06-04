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
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	api "github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/service/storage"
)

func main() {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagLevel string
		flagIndex string
		flagPort  uint16
		flagFirst uint64
	)

	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.Uint16VarP(&flagPort, "port", "p", 5005, "port to serve GRPC API on")
	pflag.Uint64VarP(&flagFirst, "first", "f", 0, "first height to fix DB with") // FIXME: remove this after all fixed

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Str("level", flagLevel).Err(err).Msg("could not parse log level")
	}
	log = log.Level(level)

	// Initialize the index core state.
	db, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Fatal().Str("index", flagIndex).Err(err).Msg("could not open index DB")
	}
	defer db.Close()

	// Check if we have a first height set and insert into DB.
	// TODO: Remove this once we have fixed all of the currently active indexes.
	// => https://github.com/optakt/flow-dps/issues/135
	if flagFirst != 0 {
		err := db.Update(storage.SaveFirst(flagFirst))
		if err != nil {
			log.Fatal().Uint64("first", flagFirst).Err(err).Msg("could not save first height")
		}
	}

	// GRPC API initialization.
	gsvr := grpc.NewServer()
	index := index.NewReader(db)
	server := api.NewServer(index)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	go func() {
		log.Info().Msg("Flow DPS Server starting")
		listener, err := net.Listen("tcp", fmt.Sprint(":", flagPort))
		if err != nil {
			log.Fatal().Uint16("port", flagPort).Err(err).Msg("could not listen")
		}
		api.RegisterAPIServer(gsvr, server)
		err = gsvr.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("GRPC API encountered error")
		}
		log.Info().Msg("Flow DPS Server stopped")
	}()

	<-sig
	log.Info().Msg("Flow DPS Server stopping")
	go func() {
		<-sig
		log.Warn().Msg("forcing exit")
		os.Exit(1)
	}()

	// The following code starts a shut down with a certain timeout and makes
	// sure that the main executing components are shutting down within the
	// allocated shutdown time. Otherwise, we will force the shutdown and log
	// an error. We then wait for shutdown on each component to complete.
	gsvr.GracefulStop()

	os.Exit(0)
}
