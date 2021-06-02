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
	"os"
	"os/signal"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/tsdb/wal"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/chain"
	"github.com/optakt/flow-dps/service/feeder"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/service/mapper"
)

func main() {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagCheckpoint string
		flagData       string
		flagIndex      string
		flagLog        string
		flagTrie       string
	)

	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "", "checkpoint file for state trie")
	pflag.StringVarP(&flagData, "data", "d", "", "database directory for protocol data")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLog, "log", "l", "info", "log output level")
	pflag.StringVarP(&flagTrie, "trie", "t", "", "data directory for state ledger")

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
	db, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Fatal().Err(err).Msg("could not open index DB")
	}
	index := index.NewWriter(db)

	// Initialize indexer components.
	data, err := badger.Open(dps.DefaultOptions(flagData))
	if err != nil {
		log.Fatal().Err(err).Msg("could not open blockchain database")
	}
	chain := chain.FromProtocolState(data)
	segments, err := wal.NewSegmentsReader(flagTrie)
	if err != nil {
		log.Fatal().Err(err).Msg("could not open segments reader")
	}
	feeder, err := feeder.FromLedgerWAL(wal.NewReader(segments))
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize feeder")
	}
	mapper, err := mapper.New(log, chain, feeder, index, mapper.WithCheckpointFile(flagCheckpoint))
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize mapper")
	}

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	go func() {
		start := time.Now()
		log.Info().Time("start", start).Msg("Flow DPS Indexer starting")
		err := mapper.Run()
		if err != nil {
			log.Error().Err(err).Msg("disk mapper encountered error")
		}
		finish := time.Now()
		duration := finish.Sub(start)
		log.Info().Time("finish", finish).Str("duration", duration.Round(time.Second).String()).Msg("Flow DPS Indexer stopped")
	}()

	select {
	case <-sig:
		log.Info().Msg("Flow DPS Indexer stopping")
	case <-mapper.Done():
		log.Info().Msg("Flow DPS Indexer done")
	}
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
	err = mapper.Stop(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not stop indexer")
	}

	os.Exit(0)
}
