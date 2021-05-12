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
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/awfm9/flow-dps/chain"
	"github.com/awfm9/flow-dps/feeder"
	grpcApi "github.com/awfm9/flow-dps/grpc"
	"github.com/awfm9/flow-dps/mapper"
	"github.com/awfm9/flow-dps/rest"
	"github.com/awfm9/flow-dps/state"
)

func main() {

	var (
		flagLevel      string
		flagData       string
		flagTrie       string
		flagIndex      string
		flagCheckpoint string
		flagHostREST   string
		flagHostGRPC   string
	)

	pflag.StringVarP(&flagLevel, "log-level", "l", "info", "log output level")
	pflag.StringVarP(&flagData, "data-dir", "d", "data", "protocol state database directory")
	pflag.StringVarP(&flagTrie, "trie-dir", "t", "trie", "state trie write-ahead log directory")
	pflag.StringVarP(&flagIndex, "index-dir", "i", "index", "state ledger index directory")
	pflag.StringVarP(&flagCheckpoint, "checkpoint-file", "c", "", "state trie root checkpoint file")
	pflag.StringVarP(&flagHostREST, "rest-host", "r", ":8080", "host URL for the REST API endpoint")
	pflag.StringVarP(&flagHostGRPC, "grpc-host", "g", ":5005", "host URL for GRPC API endpoint")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	chain, err := chain.FromProtocolState(flagData)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize chain")
	}

	feeder, err := feeder.FromLedgerWAL(flagTrie)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize feeder")
	}

	core, err := state.NewCore(flagIndex)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize ledger")
	}

	mapper, err := mapper.New(log, chain, feeder, core, mapper.WithCheckpointFile(flagCheckpoint))
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize mapper")
	}

	rctrl, err := rest.NewController(core)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize REST controller")
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Logger())
	e.GET("/registers/:key", rctrl.GetRegister)
	e.GET("/values/:keys", rctrl.GetValue)

	gctrl, err := grpcApi.NewController(core)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize GRPC controller")
	}
	grpcSrv := grpc.NewServer()

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	go func() {
		start := time.Now().UTC()
		err := mapper.Run()
		if err != nil {
			log.Error().Err(err).Msg("state mapper encountered error")
		}
		finish := time.Now().UTC()
		log.Info().Dur("duration", finish.Sub(start)).Msg("state mapper execution complete")
	}()
	go func() {
		err := e.Start(flagHostREST)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("REST API encountered error")
		}
		log.Info().Msg("REST API execution complete")
	}()
	go func() {
		lis, err := net.Listen("tcp", flagHostGRPC)
		if err != nil {
			log.Fatal().Err(err).Str("host", flagHostGRPC).Msg("could not listen")
		}

		grpcApi.RegisterAPIServer(grpcSrv, grpcApi.NewServer(gctrl))
		err = grpcSrv.Serve(lis)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("GRPC API encountered error")
		}
		log.Info().Msg("GRPC API execution complete")
	}()

	<-sig

	log.Info().Msg("startup complete")

	// The following code starts a shut down with a certain timeout and makes
	// sure that the main executing components are shutting down within the
	// allocated shutdown time. Otherwise, we will force the shutdown and log
	// an error. We then wait for shutdown on each component to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		err := e.Shutdown(ctx)
		if err != nil {
			log.Error().Err(err).Msg("could not shut down REST API")
		}
		log.Info().Msg("REST API shutdown complete")
	}()
	go func() {
		defer wg.Done()
		grpcSrv.GracefulStop()
		log.Info().Msg("state mapper shutdown complete")
	}()
	go func() {
		defer wg.Done()
		err := mapper.Stop(ctx)
		if err != nil {
			log.Error().Err(err).Msg("could not shut down state mapper")
		}
		log.Info().Msg("state mapper shutdown complete")
	}()

	wg.Wait()

	log.Info().Msg("shutdown complete")

	os.Exit(0)
}
