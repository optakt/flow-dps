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

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/ziflex/lecho/v2"
	"google.golang.org/grpc"

	api "github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/executor"
	"github.com/optakt/flow-dps/invoker"
)

const (
	success = 0
	failure = 1
)

func main() {
	os.Exit(run())
}

func run() int {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	var (
		flagAPI   string
		flagCache uint64
		flagLevel string
		flagPort  uint16
	)

	pflag.StringVarP(&flagAPI, "api", "a", "127.0.0.1:5005", "host for GRPC API server")
	pflag.Uint64VarP(&flagCache, "cache", "e", uint64(datasize.GB), "maximum cache size for register reads in bytes")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.Uint16VarP(&flagPort, "port", "p", 8080, "port to host Executor API on")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)
	elog := lecho.From(log)

	zerolog.SetGlobalLevel(level)

	// Initialize the API client.
	conn, err := grpc.Dial(flagAPI, grpc.WithInsecure())
	if err != nil {
		log.Error().Str("api", flagAPI).Err(err).Msg("could not dial API host")
		return failure
	}
	defer conn.Close()

	// Initialize storage library.
	codec, err := zbor.NewCodec()
	if err != nil {
		log.Error().Err(err).Msg("could not initialize storage codec")
		return failure
	}

	// Execute the script using remote lookup and read.
	client := api.NewAPIClient(conn)
	index := api.IndexFromAPI(client, codec)
	invoker, err := invoker.New(index, invoker.WithCacheSize(flagCache))
	if err != nil {
		log.Error().Err(err).Msg("could not initialize invoker")
		return failure
	}

	executor := executor.New(invoker)

	server := echo.New()
	server.HideBanner = true
	server.HidePort = true
	server.Logger = elog
	server.Use(lecho.Middleware(lecho.Config{Logger: elog}))
	server.POST("/execute", executor.Script)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	done := make(chan struct{})
	failed := make(chan struct{})
	go func() {
		log.Info().Msg("Flow Executor starting")
		err := server.Start(fmt.Sprint(":", flagPort))
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn().Err(err).Msg("Flow Executor failed")
			close(failed)
		} else {
			close(done)
		}
		log.Info().Msg("Flow Executor stopped")
	}()

	select {
	case <-sig:
		log.Info().Msg("Flow Executor stopping")
	case <-done:
		log.Info().Msg("Flow Executor done")
	case <-failed:
		log.Warn().Msg("Flow Executor aborted")
		return failure
	}
	go func() {
		<-sig
		log.Warn().Msg("forcing exit")
		os.Exit(1)
	}()

	return success
}
