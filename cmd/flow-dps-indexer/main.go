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
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/tsdb/wal"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/chain"
	"github.com/optakt/flow-dps/service/feeder"
	"github.com/optakt/flow-dps/service/forest"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/service/initializer"
	"github.com/optakt/flow-dps/service/mapper"
	"github.com/optakt/flow-dps/service/storage"
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
		flagCheckpoint string
		flagData       string
		flagForce      bool
		flagIndex      string
		flagLevel      string
		flagSkip       bool
		flagTrie       string
	)

	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "root.checkpoint", "checkpoint file for state trie")
	pflag.StringVarP(&flagData, "data", "d", "", "database directory for protocol data")
	pflag.BoolVarP(&flagForce, "force", "f", false, "overwrite existing index database")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.BoolVarP(&flagSkip, "skip", "s", false, "skip indexing of ledger registers")
	pflag.StringVarP(&flagTrie, "trie", "t", "", "data directory for state ledger")

	pflag.Parse()

	// Increase the GOMAXPROCS value in order to use the full IOPS available, see:
	// https://groups.google.com/g/golang-nuts/c/jPb_h3TvlKE
	_ = runtime.GOMAXPROCS(128)

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Err(err).Str("level", flagLevel).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	// The root checkpoint is mandatory, so check if we can load it first.
	root, err := initializer.RootTrie(flagCheckpoint)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize root trie")
		return failure
	}

	// Open index database.
	db, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Error().Err(err).Str("index", flagIndex).Msg("could not open index DB")
		return failure
	}
	defer db.Close()

	// Open protocol state database.
	data, err := badger.Open(dps.DefaultOptions(flagData))
	if err != nil {
		log.Error().Err(err).Msg("could not open blockchain database")
		return failure
	}
	defer data.Close()

	// Initialize a codec and a storage function library that is used by the
	// index reader and writer to manage index database access.
	codec := zbor.NewCodec()
	storage := storage.New(codec)
	read := index.NewReader(db, storage)
	write := index.NewWriter(db, storage)
	defer func() {
		err := write.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close index writer")
		}
	}()

	// The chain is responsible for reading blockchain data from the protocol state.
	disk := chain.FromDisk(data)

	// Feeder is responsible for reading the write-ahead log of the execution state.
	segments, err := wal.NewSegmentsReader(flagTrie)
	if err != nil {
		log.Error().Err(err).Str("trie", flagTrie).Msg("could not open segments reader")
		return failure
	}
	feed := feeder.FromWAL(wal.NewReader(segments))

	// Writer is responsible for writing the index data to the index database.
	index := index.NewWriter(db, storage)
	defer func() {
		err := index.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close index")
		}
	}()

	// Initialize the transitions with the dependencies and add them to the FSM.
	transitions := mapper.NewTransitions(log, disk, feed, read, write,
		mapper.WithRootTrie(root),
		mapper.WithSkipRegisters(flagSkip),
	)
	forest := forest.New()
	state := mapper.EmptyState(forest)
	fsm := mapper.NewFSM(state,
		mapper.WithTransition(mapper.StatusBootstrap, transitions.BootstrapState),
		mapper.WithTransition(mapper.StatusIndex, transitions.IndexChain),
		mapper.WithTransition(mapper.StatusUpdate, transitions.UpdateTree),
		mapper.WithTransition(mapper.StatusCollect, transitions.CollectRegisters),
		mapper.WithTransition(mapper.StatusMap, transitions.MapRegisters),
		mapper.WithTransition(mapper.StatusForward, transitions.ForwardHeight),
	)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	done := make(chan struct{})
	failed := make(chan struct{})
	go func() {
		start := time.Now()
		log.Info().Time("start", start).Msg("Flow DPS Indexer starting")
		err := fsm.Run()
		if err != nil {
			log.Warn().Err(err).Msg("Flow DPS Indexer failed")
			close(failed)
		} else {
			close(done)
		}
		finish := time.Now()
		duration := finish.Sub(start)
		log.Info().Time("finish", finish).Str("duration", duration.Round(time.Second).String()).Msg("Flow DPS Indexer stopped")
	}()

	select {
	case <-sig:
		log.Info().Msg("Flow DPS Indexer stopping")
	case <-done:
		log.Info().Msg("Flow DPS Indexer done")
	case <-failed:
		log.Warn().Msg("Flow DPS Indexer aborted")
		return failure
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
	err = fsm.Stop()
	if err != nil {
		log.Error().Err(err).Msg("could not stop indexer")
		return failure
	}

	return success
}
