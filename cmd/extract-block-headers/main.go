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
	"io/ioutil"
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
)

func main() {

	// Command line parameter initialization.
	var (
		flagLevel  string
		flagData   string
		flagBegin  uint64
		flagFinish uint64
		flagOutput string
		flagSize   uint64
	)

	pflag.StringVarP(&flagLevel, "log-level", "l", "info", " log level for JSON logger output")
	pflag.StringVarP(&flagData, "data-dir", "d", "data", "directory for protocol state database")
	pflag.Uint64VarP(&flagBegin, "begin-height", "b", 0, "lowest block height to include in extraction")
	pflag.Uint64VarP(&flagFinish, "finish-height", "f", 100_000_000, "highest block height to include in extraction")
	pflag.StringVarP(&flagOutput, "output-dir", "o", "headers", "directory for output of ledger payloads")
	pflag.Uint64VarP(&flagSize, "size-limit", "s", 11_264_000, "limit for total size of output files")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// Initialize the protocol state database we will use.
	opts := badger.DefaultOptions(flagData).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("could not open database")
	}

	// Initialize the codec we use for the data.
	codec, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize codec")
	}

	// Make a list of all available heights and shufftle them.
	var heights []uint64
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
			log.Fatal().Err(err).Msg("could not look up block")
		}
		var header flow.Header
		err = db.View(operation.RetrieveHeader(blockID, &header))
		if err != nil {
			log.Fatal().Err(err).Msg("could not retrieve header")
		}
		data, err := codec.Marshal(&header)
		if err != nil {
			log.Fatal().Err(err).Msg("could not encode header")
		}
		name := filepath.Join(flagOutput, fmt.Sprintf("header-%07d", index))
		err = ioutil.WriteFile(name, data, fs.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msg("could not write header file")
		}
		total += uint64(len(data))
		log.Info().Int("header_size", len(data)).Uint64("total_size", total).Msg("block header extracted")
		if total > flagSize {
			break
		}
	}

	os.Exit(0)
}
