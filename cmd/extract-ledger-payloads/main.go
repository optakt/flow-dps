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
	"fmt"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/prometheus/tsdb/wal"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"

	"github.com/optakt/flow-dps/service/chain"
	"github.com/optakt/flow-dps/service/feeder"
	"github.com/optakt/flow-dps/service/mapper"

	"github.com/optakt/flow-dps/models/dps"
)

func main() {

	// Command line parameter initialization.
	var (
		flagCheckpoint string
		flagData       string
		flagLog        string
		flagOutput     string
		flagSize       uint64
		flagTrie       string
	)

	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "root.checkpoint", "file containing state trie snapshot")
	pflag.StringVarP(&flagData, "data", "d", "data", "directory for protocol state database")
	pflag.StringVarP(&flagLog, "log", "l", "info", "log level for JSON logger output")
	pflag.StringVarP(&flagOutput, "output", "o", "payloads", "directory for output of ledger payloads")
	pflag.Uint64VarP(&flagSize, "size", "s", 11_264_000, "limit for total size of output files")
	pflag.StringVarP(&flagTrie, "trie", "t", "trie", "directory for execution state database")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLog)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// Set up the closure to capture the tree after processing finished.
	var tree *trie.MTrie
	post := func(t *trie.MTrie) {
		tree = t
	}

	// Initialize the mapper.
	opts := dps.DefaultOptions(flagData).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("could not open blockchain database")
	}

	chain := chain.FromProtocolState(db)

	segments, err := wal.NewSegmentsReader(flagTrie)
	if err != nil {
		log.Fatal().Err(err).Msg("could not open segments reader")
	}
	feeder, err := feeder.FromLedgerWAL(wal.NewReader(segments))
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize feeder")
	}
	mapper, err := mapper.New(log, chain, feeder, &Index{},
		mapper.WithCheckpointFile(flagCheckpoint),
		mapper.WithPostProcessing(post),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize mapper")
	}

	log.Info().Msg("starting disk mapper to build final state trie")

	// Run the mapper to get the latest trie.
	start := time.Now()
	err = mapper.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("disk mapper encountered error")
	}
	finish := time.Now()
	delta := finish.Sub(start)

	log.Info().Str("duration", delta.Round(delta).String()).Msg("disk mapper execution complete")

	// Now, we got the full trie and we can write random payloads to disk until
	// we have enough data for the dictionary creator.
	codec, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize codec")
	}
	payloads := tree.AllPayloads()
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(payloads), func(i int, j int) {
		payloads[i], payloads[j] = payloads[j], payloads[i]
	})
	total := uint64(0)
	for index, payload := range payloads {
		log := log.With().Int("index", index).Hex("key", payload.Key.CanonicalForm()).Logger()
		data, err := codec.Marshal(&payload)
		if err != nil {
			log.Fatal().Err(err).Msg("could not encode payload")
		}
		name := filepath.Join(flagOutput, fmt.Sprintf("payload-%07d", index))
		err = ioutil.WriteFile(name, data, fs.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msg("could not write payload file")
		}
		total += uint64(len(data))
		log.Info().Int("payload_size", len(data)).Uint64("total_size", total).Msg("ledger payload extracted")
		if total > flagSize {
			break
		}
	}

	os.Exit(0)
}
