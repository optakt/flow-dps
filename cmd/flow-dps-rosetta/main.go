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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/api/server"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/height"
	"github.com/optakt/flow-dps/rosetta/invoker"
	"github.com/optakt/flow-dps/rosetta/lookup"
	"github.com/optakt/flow-dps/rosetta/read"
	"github.com/optakt/flow-dps/rosetta/retriever"
	"github.com/optakt/flow-dps/rosetta/scripts"
	"github.com/optakt/flow-dps/rosetta/validator"
)

func main() {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagAPI   string
		flagChain string
		flagLog   string
		flagPort  uint16
	)

	pflag.StringVarP(&flagAPI, "api", "a", "127.0.0.1:5005", "host URL for GRPC API endpoint")
	pflag.StringVarP(&flagChain, "chain", "c", dps.FlowTestnet.String(), "specify chain ID for Flow network")
	pflag.StringVarP(&flagLog, "log", "l", "info", "log output level")
	pflag.Uint16VarP(&flagPort, "port", "p", 8080, "port to host Rosetta API on")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLog)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// Check if the configured chain ID is valid.
	params, ok := dps.FlowParams[flow.ChainID(flagChain)]
	if !ok {
		log.Fatal().Str("chain", flagChain).Msg("invalid chain ID for params")
	}

	// Initialize the API client.
	conn, err := grpc.Dial(flagAPI, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Msg("could not dial API host")
	}
	client := server.NewAPIClient(conn)

	// Rosetta API initialization.
	generator := scripts.NewGenerator(params)
	invoke := invoker.New(lookup.FromDPS(client), read.FromDPS(client))
	validate := validator.New(params, height.FromDPS(client))
	retrieve := retriever.New(generator, invoke)
	ctrl := rosetta.NewData(validate, retrieve)

	svr := echo.New()
	svr.HideBanner = true
	svr.HidePort = true
	svr.Use(middleware.Logger())
	svr.POST("/account/balance", ctrl.Balance)
	svr.POST("/block", ctrl.Block)
	svr.POST("/block/transaction", ctrl.Transaction)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	go func() {
		log.Info().Msg("Flow DPS Rosetta starting")
		err := svr.Start(fmt.Sprintf(":%d", flagPort))
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("Rosetta API encountered error")
		}
		log.Info().Msg("Flow DPS Rosetta stopped")
	}()

	<-sig
	log.Info().Msg("Flow DPS Rosetta stopping")
	go func() {
		<-sig
		log.Warn().Msg("forcing exit")
		os.Exit(1)
	}()

	// The following code starts a shut down with a certain timeout and makes
	// sure that the main executing components are shutting down within the
	// allocated shutdown time. Otherwise, we will force the shutdown and log
	// an error. We then wait for shutdown on each component to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = svr.Shutdown(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not shut down Rosetta API")
	}

	os.Exit(0)
}
