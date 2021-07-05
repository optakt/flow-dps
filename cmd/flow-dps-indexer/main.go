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
	"context"
	"os"
	"os/signal"
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
		flagCheckpoint        string
		flagData              string
		flagForce             bool
		flagIndex             string
		flagIndexAll          bool
		flagIndexCollections  bool
		flagIndexCommit       bool
		flagIndexEvents       bool
		flagIndexHeader       bool
		flagIndexPayloads     bool
		flagIndexTransactions bool
		flagLevel             string
		flagTrie              string
	)

	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "", "checkpoint file for state trie")
	pflag.StringVarP(&flagData, "data", "d", "", "database directory for protocol data")
	pflag.BoolVarP(&flagForce, "force", "f", false, "overwrite existing index database")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.BoolVarP(&flagIndexAll, "index-all", "a", false, "index everything")
	pflag.BoolVarP(&flagIndexCollections, "index-collections", "o", false, "index collections")
	pflag.BoolVarP(&flagIndexCommit, "index-commits", "m", false, "index commits")
	pflag.BoolVarP(&flagIndexEvents, "index-events", "e", false, "index events")
	pflag.BoolVarP(&flagIndexHeader, "index-headers", "h", false, "index headers")
	pflag.BoolVarP(&flagIndexPayloads, "index-payloads", "p", false, "index payloads")
	pflag.BoolVarP(&flagIndexTransactions, "index-transactions", "x", false, "index transactions")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagTrie, "trie", "t", "", "data directory for state ledger")

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

	// Ensure that at least one index is specified.
	if !flagIndexAll && !flagIndexCommit && !flagIndexEvents && !flagIndexHeader &&
		!flagIndexPayloads && !flagIndexTransactions && !flagIndexCollections {
		log.Error().Str("level", flagLevel).Msg("no indexing option specified, use -a/--all to build all indexes")
		pflag.Usage()
		return failure
	}

	// Fail if IndexAll is specified along with other index flags, as this would most likely mean that the user does
	// not understand what they are doing.
	if flagIndexAll && (flagIndexCommit || flagIndexEvents || flagIndexHeader ||
		flagIndexPayloads || flagIndexTransactions || flagIndexCollections) {
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

	// Initialize storage library.
	codec, err := zbor.NewCodec()
	if err != nil {
		log.Error().Err(err).Msg("could not initialize storage codec")
		return failure
	}
	storage := storage.New(codec)

	// Check if index already exists.
	_, err = index.NewReader(db, storage).First()
	indexExists := err == nil
	if indexExists && !flagForce {
		log.Error().Err(err).Msg("index already exists, manually delete it or use (-f, --force) to overwrite it")
		return failure
	}

	// Initialize the dependencies needed for the FSM and the state transitions.
	load := loader.New(
		loader.WithCheckpointPath(flagCheckpoint),
	)
	chain := chain.FromDisk(data)
	segments, err := wal.NewSegmentsReader(flagTrie)
	if err != nil {
		log.Error().Str("trie", flagTrie).Err(err).Msg("could not open segments reader")
		return failure
	}
	feed, err := feeder.FromDisk(wal.NewReader(segments))
	if err != nil {
		log.Error().Str("trie", flagTrie).Err(err).Msg("could not initialize feeder")
		return failure
	}
	index := index.NewWriter(db, storage)

	// Initialize the transitions with the dependencies and add them to the FSM.
	transitions := mapper.NewTransitions(log, load, chain, feed, index,
		mapper.WithIndexCommit(flagIndexAll || flagIndexCommit),
		mapper.WithIndexHeader(flagIndexAll || flagIndexHeader),
		mapper.WithIndexCollections(flagIndexAll || flagIndexCollections),
		mapper.WithIndexTransactions(flagIndexAll || flagIndexTransactions),
		mapper.WithIndexEvents(flagIndexAll || flagIndexEvents),
		mapper.WithIndexPayloads(flagIndexAll || flagIndexPayloads),
	)
	forest := forest.New()
	state := mapper.EmptyState(forest)
	fsm := mapper.NewFSM(state,
		mapper.WithTransition(mapper.StatusEmpty, transitions.BootstrapState),
		mapper.WithTransition(mapper.StatusUpdating, transitions.UpdateTree),
		mapper.WithTransition(mapper.StatusMatched, transitions.CollectRegisters),
		mapper.WithTransition(mapper.StatusCollected, transitions.IndexRegisters),
		mapper.WithTransition(mapper.StatusIndexed, transitions.ForwardHeight),
		mapper.WithTransition(mapper.StatusForwarded, transitions.IndexChain),
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = fsm.Stop(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not stop indexer")
		return failure
	}

	return success
}
