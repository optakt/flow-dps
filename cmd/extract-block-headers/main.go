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
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

const (
	success = 0
	failure = 1
)

func main() {
	os.Exit(run())
}

func run() int {

	// Command line parameter initialization.
	var (
		flagBegin  uint64
		flagData   string
		flagFinish uint64
		flagLevel  string
		flagOutput string
		flagSize   uint64
	)

	pflag.Uint64VarP(&flagBegin, "begin", "b", 0, "lowest block height to include in extraction")
	pflag.StringVarP(&flagData, "data", "d", "data", "directory for protocol state database")
	pflag.Uint64VarP(&flagFinish, "finish", "f", 100_000_000, "highest block height to include in extraction")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log level for JSON logger output")
	pflag.StringVarP(&flagOutput, "output", "o", "headers", "directory for output of block headers")
	pflag.Uint64VarP(&flagSize, "size", "s", 11_264_000, "limit for total size of output files")

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

	// Initialize the protocol state database we will use.
	opts := dps.DefaultOptions(flagData).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		log.Error().Str("data", flagData).Err(err).Msg("could not open blockchain database")
		return failure
	}
	defer db.Close()

	// Initialize the codec we use for the data.
	codec, _ := cbor.CanonicalEncOptions().EncMode()

	// Make a list of all available heights and shuffle them.
	if flagBegin > flagFinish {
		flagBegin, flagFinish = flagFinish, flagBegin
	}
	heights := make([]uint64, 0, flagFinish-flagBegin)
	for height := flagBegin; height <= flagFinish; height++ {
		heights = append(heights, height)
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(heights), func(i int, j int) {
		heights[i], heights[j] = heights[j], heights[i]
	})

	// Go through heights, try to get the block on each height until we reach
	// the end or the maximum configured size.
	total := uint64(0)
	for index, height := range heights {
		log := log.With().Int("index", index).Uint64("height", height).Logger()
		var blockID flow.Identifier
		err = db.View(operation.LookupBlockHeight(height, &blockID))
		if errors.Is(err, storage.ErrNotFound) {
			log.Warn().Err(err).Msg("invalid block height")
			continue
		}
		if err != nil {
			log.Error().Err(err).Msg("could not look up block")
			return failure
		}
		var header flow.Header
		err = db.View(operation.RetrieveHeader(blockID, &header))
		if err != nil {
			log.Error().Err(err).Msg("could not retrieve header")
			return failure
		}
		data, err := codec.Marshal(&header)
		if err != nil {
			log.Error().Err(err).Msg("could not encode header")
			return failure
		}
		name := filepath.Join(flagOutput, fmt.Sprintf("header-%07d", index))
		err = os.WriteFile(name, data, fs.ModePerm)
		if err != nil {
			log.Error().Err(err).Msg("could not write header file")
			return failure
		}
		total += uint64(len(data))
		log.Info().Int("header_size", len(data)).Uint64("total_size", total).Msg("block header extracted")
		if total > flagSize {
			break
		}
	}

	return success
}
