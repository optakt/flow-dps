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
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/tsdb/wal"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/engine"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/chain"
	"github.com/optakt/flow-dps/service/feeder"
	"github.com/optakt/flow-dps/service/forest"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/service/loader"
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
		flagIndex      string
		flagLevel      string
		flagTrie       string
		flagSkip       bool
	)

	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "", "path to root checkpoint file for execution state trie")
	pflag.StringVarP(&flagData, "data", "d", "data", "path to database directory for protocol data")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "path to database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagTrie, "trie", "t", "", "path to data directory for execution state ledger")
	pflag.BoolVarP(&flagSkip, "skip", "s", false, "skip indexing of execution state ledger registers")

	pflag.Parse()

	// Increase the GOMAXPROCS value in order to use the full IOPS available, see:
	// https://groups.google.com/g/golang-nuts/c/jPb_h3TvlKE
	_ = runtime.GOMAXPROCS(128)

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	// Open the needed databases.
	indexDB, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open index database")
		return failure
	}
	defer func() {
		err := indexDB.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close index database")
		}
	}()
	protocolDB, err := badger.Open(dps.DefaultOptions(flagData))
	if err != nil {
		log.Error().Err(err).Msg("could not open protocol state database")
		return failure
	}
	defer func() {
		err := protocolDB.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close protocol state database")
		}
	}()

	// The storage library is initialized with a codec and provides functions to
	// interact with a Badger database while encoding and compressing
	// transparently.
	codec := zbor.NewCodec()
	storage := storage.New(codec)

	// Check if index already exists.
	read := index.NewReader(indexDB, storage)
	_, err = read.First()
	empty := errors.Is(err, badger.ErrKeyNotFound)
	if err != nil && !empty {
		log.Error().Err(err).Msg("could not get first height from index reader")
		return failure
	}
	if empty && flagCheckpoint == "" {
		log.Error().Msg("index doesn't exist, please provide root checkpoint (-c, --checkpoint) to bootstrap")
		return failure
	}

	// The chain is responsible for reading blockchain data from the protocol state.
	disk := chain.FromDisk(protocolDB)

	// Feeder is responsible for reading the write-ahead log of the execution state.
	segments, err := wal.NewSegmentsReader(flagTrie)
	if err != nil {
		log.Error().Str("trie", flagTrie).Err(err).Msg("could not open segments reader")
		return failure
	}
	feed := feeder.FromWAL(wal.NewReader(segments))

	// Writer is responsible for writing the index data to the index database.
	// We explicitly disable flushing at regular intervals to improve throughput
	// of badger transactions when indexing from static on-disk data.
	write := index.NewWriter(indexDB, storage,
		index.WithFlushInterval(0),
	)
	defer func() {
		err := write.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close index")
		}
	}()

	// Initialize the transitions with the dependencies and add them to the FSM.
	load := loader.FromScratch()

	transitions := mapper.NewTransitions(log, load, disk, feed, read, write,
		mapper.WithBootstrapState(true),
		mapper.WithSkipRegisters(flagSkip),
	)
	forest := forest.New()
	state := mapper.EmptyState(forest)
	fsm := mapper.NewFSM(state,
		mapper.WithTransition(mapper.StatusInitialize, transitions.InitializeMapper),
		mapper.WithTransition(mapper.StatusBootstrap, transitions.BootstrapState),
		mapper.WithTransition(mapper.StatusResume, transitions.ResumeIndexing),
		mapper.WithTransition(mapper.StatusIndex, transitions.IndexChain),
		mapper.WithTransition(mapper.StatusUpdate, transitions.UpdateTree),
		mapper.WithTransition(mapper.StatusCollect, transitions.CollectRegisters),
		mapper.WithTransition(mapper.StatusMap, transitions.MapRegisters),
		mapper.WithTransition(mapper.StatusForward, transitions.ForwardHeight),
	)

	err = engine.New(log, "Flow DPS Indexer", sig).
		Component(
			"mapper",
			func() error {
				return fsm.Run()
			},
			func() {
				fsm.Stop()
			},
		).
		Run()
	if err != nil {
		log.Error().Err(err).Msg("failed")
		return failure
	}

	return success
}
