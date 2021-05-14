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
	gsvr "google.golang.org/grpc"

	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/api/grpc"
	"github.com/awfm9/flow-dps/api/rest"
	"github.com/awfm9/flow-dps/api/rosetta"

	"github.com/awfm9/flow-dps/rosetta/contracts"
	"github.com/awfm9/flow-dps/rosetta/invoker"
	"github.com/awfm9/flow-dps/rosetta/retriever"
	"github.com/awfm9/flow-dps/rosetta/scripts"
	"github.com/awfm9/flow-dps/rosetta/validator"

	"github.com/awfm9/flow-dps/service/chain"
	"github.com/awfm9/flow-dps/service/feeder"
	"github.com/awfm9/flow-dps/service/mapper"
	"github.com/awfm9/flow-dps/service/state"
)

func main() {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagLevel       string
		flagData        string
		flagTrie        string
		flagIndex       string
		flagCheckpoint  string
		flagHostREST    string
		flagHostGRPC    string
		flagHostRosetta string
		flagFlowToken   string
	)

	pflag.StringVarP(&flagLevel, "log-level", "l", "info", "log output level")
	pflag.StringVarP(&flagData, "data-dir", "d", "data", "protocol state database directory")
	pflag.StringVarP(&flagTrie, "trie-dir", "t", "trie", "state trie write-ahead log directory")
	pflag.StringVarP(&flagIndex, "index-dir", "i", "index", "state ledger index directory")
	pflag.StringVarP(&flagCheckpoint, "checkpoint-file", "c", "", "state trie root checkpoint file")
	pflag.StringVarP(&flagHostREST, "rest-host", "r", ":8080", "host URL for REST API endpoints")
	pflag.StringVarP(&flagHostGRPC, "grpc-host", "g", ":5005", "host URL for GRPC API endpoints")
	pflag.StringVarP(&flagHostRosetta, "rosetta-host", "a", ":8090", "host UR for Rosetta endpoints")
	pflag.StringVarP(&flagFlowToken, "flow-token", "f", "0x7e60df042a9c0868", "address of the Flow token contract")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// DPS indexer initialization.
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
	mapper, err := mapper.New(log, chain, feeder, core.Index(), mapper.WithCheckpointFile(flagCheckpoint))
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize mapper")
	}

	// REST API initialization.
	rctrl, err := rest.NewController(core)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize REST controller")
	}

	rsvr := echo.New()
	rsvr.HideBanner = true
	rsvr.HidePort = true
	rsvr.Use(middleware.Logger())
	rsvr.GET("/registers/:key", rctrl.GetRegister)
	rsvr.GET("/values/:keys", rctrl.GetValue)

	// GRPC API initialization.
	gctrl, err := grpc.NewController(core)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize GRPC controller")
	}
	gsvr := gsvr.NewServer()

	// Rosetta API initialization.
	contracts := contracts.New(contracts.WithToken("FLOW", flow.HexToAddress(flagFlowToken)))
	scripts := scripts.New(scripts.WithParams(scripts.TestNet()))
	invoke := invoker.New(log, core)
	validate := validator.New(core.Height(), contracts)
	retrieve := retriever.New(contracts, scripts, invoke)
	actrl := rosetta.NewData(validate, retrieve)

	asvr := echo.New()
	asvr.HideBanner = true
	asvr.HidePort = true
	asvr.Use(middleware.Logger())
	asvr.POST("/account/balance", actrl.Balance)
	asvr.POST("/block", actrl.Block)
	asvr.POST("/block/transaction", actrl.Transaction)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	go func() {
		start := time.Now().UTC()
		err := mapper.Run()
		if err != nil {
			log.Error().Err(err).Msg("disk mapper encountered error")
		}
		finish := time.Now().UTC()
		log.Info().Dur("duration", finish.Sub(start)).Msg("disk mapper execution complete")
	}()
	go func() {
		err := rsvr.Start(flagHostREST)
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
		grpc.RegisterAPIServer(gsvr, grpc.NewServer(gctrl))
		err = gsvr.Serve(lis)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("GRPC API encountered error")
		}
		log.Info().Msg("GRPC API execution complete")
	}()
	go func() {
		err := asvr.Start(flagHostRosetta)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("Rosetta API encountered error")
		}
		log.Info().Msg("Rosetta API execution complete")
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
	wg.Add(4)
	go func() {
		defer wg.Done()
		err := asvr.Shutdown(ctx)
		if err != nil {
			log.Error().Err(err).Msg("could not shut down Rosetta API")
		}
		log.Info().Msg("Rosetta API shutdown complete")
	}()
	go func() {
		defer wg.Done()
		gsvr.GracefulStop()
		log.Info().Msg("GRPC API shutdown complete")
	}()
	go func() {
		defer wg.Done()
		err := rsvr.Shutdown(ctx)
		if err != nil {
			log.Error().Err(err).Msg("could not shut down REST API")
		}
		log.Info().Msg("REST API shutdown complete")
	}()
	go func() {
		defer wg.Done()
		err := mapper.Stop(ctx)
		if err != nil {
			log.Error().Err(err).Msg("could not shut down mapper")
		}
		log.Info().Msg("disk mapper shutdown complete")
	}()

	wg.Wait()

	log.Info().Msg("shutdown complete")

	os.Exit(0)
}
