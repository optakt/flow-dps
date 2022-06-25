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

	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/service/chain"
	"github.com/onflow/flow-dps/service/feeder"
	"github.com/onflow/flow-dps/service/forest"
	"github.com/onflow/flow-dps/service/index"
	"github.com/onflow/flow-dps/service/loader"
	"github.com/onflow/flow-dps/service/mapper"
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
	first, err := read.First()
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
	var load mapper.Loader
	load = loader.FromIndex(log, storage, indexDB)
	bootstrap := flagCheckpoint != ""
	if empty {
		file, err := os.Open(flagCheckpoint)
		if err != nil {
			log.Error().Err(err).Msg("could not open checkpoint file")
			return failure
		}
		file.Close()
		load = loader.FromCheckpointFile(flagCheckpoint, &log)
	} else if bootstrap {
		file, err := os.Open(flagCheckpoint)
		if err != nil {
			log.Error().Err(err).Msg("could not open checkpoint file")
			return failure
		}
		file.Close()
		initialize := loader.FromCheckpointFile(flagCheckpoint, &log)
		load = loader.FromIndex(log, storage, indexDB,
			loader.WithInitializer(initialize),
			loader.WithExclude(loader.ExcludeAtOrBelow(first)),
		)
	}

	transitions := mapper.NewTransitions(log, load, disk, feed, read, write,
		mapper.WithBootstrapState(bootstrap),
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
