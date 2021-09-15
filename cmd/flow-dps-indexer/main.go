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
		flagCheckpoint        string
		flagData              string
		flagForce             bool
		flagIndex             string
		flagIndexAll          bool
		flagIndexCollections  bool
		flagIndexGuarantees   bool
		flagIndexCommit       bool
		flagIndexEvents       bool
		flagIndexHeader       bool
		flagIndexPayloads     bool
		flagIndexResults      bool
		flagIndexTransactions bool
		flagIndexSeals        bool
		flagLevel             string
		flagTrie              string
	)

	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "", "checkpoint file for state trie")
	pflag.StringVarP(&flagData, "data", "d", "", "database directory for protocol data")
	pflag.BoolVarP(&flagForce, "force", "f", false, "overwrite existing index database")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.BoolVarP(&flagIndexAll, "index-all", "a", false, "index everything")
	pflag.BoolVar(&flagIndexCollections, "index-collections", false, "index collections")
	pflag.BoolVar(&flagIndexGuarantees, "index-guarantees", false, "index collection guarantees")
	pflag.BoolVar(&flagIndexCommit, "index-commits", false, "index commits")
	pflag.BoolVar(&flagIndexEvents, "index-events", false, "index events")
	pflag.BoolVar(&flagIndexHeader, "index-headers", false, "index headers")
	pflag.BoolVar(&flagIndexPayloads, "index-payloads", false, "index payloads")
	pflag.BoolVar(&flagIndexResults, "index-results", false, "index transaction results")
	pflag.BoolVar(&flagIndexTransactions, "index-transactions", false, "index transactions")
	pflag.BoolVar(&flagIndexSeals, "index-seals", false, "index seals")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
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
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	// Ensure that at least one index is specified.
	if !flagIndexAll && !flagIndexCommit && !flagIndexHeader && !flagIndexPayloads && !flagIndexCollections &&
		!flagIndexGuarantees && !flagIndexTransactions && !flagIndexResults && !flagIndexEvents && !flagIndexSeals {
		log.Error().Str("level", flagLevel).Msg("no indexing option specified, use -a/--all to build all indexes")
		pflag.Usage()
		return failure
	}

	// Fail if IndexAll is specified along with other index flags, as this would most likely mean that the user does
	// not understand what they are doing.
	if flagIndexAll && (flagIndexCommit || flagIndexHeader || flagIndexPayloads || flagIndexGuarantees ||
		flagIndexCollections || flagIndexTransactions || flagIndexResults || flagIndexEvents || flagIndexSeals) {
		log.Error().Str("level", flagLevel).Msg("-a/--all is mutually exclusive with specific indexing flags")
		pflag.Usage()
		return failure
	}

	// Open index database.
	db, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open index DB")
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

	// The storage library is initialized with a codec and provides functions to
	// interact with a Badger database while encoding and compressing
	// transparently.
	codec := zbor.NewCodec()
	storage := storage.New(codec)

	// Check if index already exists.
	_, err = index.NewReader(db, storage).First()
	indexExists := err == nil
	if indexExists && !flagForce {
		log.Error().Err(err).Msg("index already exists, manually delete it or use (-f, --force) to overwrite it")
		return failure
	}

	// Load the root trie from the checkpoint file providen on command line.
	root, err := initializer.RootTrie(flagCheckpoint)
	if err != nil {
		log.Error().Err(err).Msg("could not load root checkpoint")
		return failure
	}

	// The chain is responsible for reading blockchain data from the protocol state.
	disk := chain.FromDisk(data)

	// Feeder is responsible for reading the write-ahead log of the execution state.
	segments, err := wal.NewSegmentsReader(flagTrie)
	if err != nil {
		log.Error().Str("trie", flagTrie).Err(err).Msg("could not open segments reader")
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
	write := dps.Writer(index)

	// Initialize the transitions with the dependencies and add them to the FSM.
	transitions := mapper.NewTransitions(log, root, disk, feed, write,
		mapper.WithIndexCommit(flagIndexAll || flagIndexCommit),
		mapper.WithIndexHeader(flagIndexAll || flagIndexHeader),
		mapper.WithIndexCollections(flagIndexAll || flagIndexCollections),
		mapper.WithIndexGuarantees(flagIndexAll || flagIndexGuarantees),
		mapper.WithIndexTransactions(flagIndexAll || flagIndexTransactions),
		mapper.WithIndexResults(flagIndexAll || flagIndexResults),
		mapper.WithIndexEvents(flagIndexAll || flagIndexEvents),
		mapper.WithIndexPayloads(flagIndexAll || flagIndexPayloads),
		mapper.WithIndexSeals(flagIndexAll || flagIndexSeals),
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
