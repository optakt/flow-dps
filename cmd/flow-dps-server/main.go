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
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	grpczerolog "github.com/grpc-ecosystem/go-grpc-middleware/providers/zerolog/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"

	api "github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/service/index"
	"github.com/onflow/flow-dps/service/storage"
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

	// Command line parameter initialization.
	var (
		flagAddress string
		flagLevel   string
		flagIndex   string
	)

	pflag.StringVarP(&flagAddress, "address", "a", "127.0.0.1:5005", "bind address for serving DPS API")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "path to database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")

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

	// Initialize the index core state and open database in read-only mode.
	db, err := badger.Open(dps.DefaultOptions(flagIndex).WithReadOnly(true))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open index DB")
		return failure
	}
	defer db.Close()

	// Initialize storage library.
	codec := zbor.NewCodec()
	storage := storage.New(codec)

	// GRPC API initialization.
	opts := []logging.Option{
		logging.WithLevels(logging.DefaultServerCodeToLevel),
	}
	gsvr := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			tags.UnaryServerInterceptor(),
			logging.UnaryServerInterceptor(grpczerolog.InterceptorLogger(log), opts...),
		),
		grpc.ChainStreamInterceptor(
			tags.StreamServerInterceptor(),
			logging.StreamServerInterceptor(grpczerolog.InterceptorLogger(log), opts...),
		),
	)
	index := index.NewReader(db, storage)
	server := api.NewServer(index, codec)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	listener, err := net.Listen("tcp", flagAddress)
	if err != nil {
		log.Error().Str("address", flagAddress).Err(err).Msg("could not create listener")
		return failure
	}
	done := make(chan struct{})
	failed := make(chan struct{})
	go func() {
		log.Info().Msg("Flow DPS Server starting")
		api.RegisterAPIServer(gsvr, server)
		err = gsvr.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn().Err(err).Msg("Flow DPS Server failed")
			close(failed)
		} else {
			close(done)
		}
		log.Info().Msg("Flow DPS Server stopped")
	}()

	select {
	case <-sig:
		log.Info().Msg("Flow DPS Server stopping")
	case <-done:
		log.Info().Msg("Flow DPS Server done")
	case <-failed:
		log.Warn().Msg("Flow DPS Server aborted")
		return failure
	}
	go func() {
		<-sig
		log.Warn().Msg("forcing exit")
		os.Exit(1)
	}()

	gsvr.GracefulStop()

	return success
}
