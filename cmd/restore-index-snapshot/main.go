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
	"compress/gzip"
	"encoding/hex"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/optakt/flow-dps/codec/zbor"
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

	// Parse the command line arguments.
	var (
		flagIndex  string
		flagLevel  string
		flagFormat string
	)

	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagFormat, "format", "f", "hex", "input format (hex, gzip or raw)")

	pflag.Parse()

	// Initialize the logger.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	// Validate input format
	switch flagFormat {
	case "hex", "gzip", "raw":
		// Valid input formats.
	default:
		log.Error().Msg("invalid input format")
		return failure
	}

	// Create input reader
	var reader io.Reader
	reader = os.Stdin
	if flagFormat == "hex" {
		reader = hex.NewDecoder(reader)

	} else if flagFormat == "gzip" {

		// Create a gzip reader.
		gzr, err := gzip.NewReader(reader)
		if err != nil {
			log.Error().Err(err).Msg("could not create gzip reader")
			return failure
		}

		// Log the snapshot metadata.
		log.Info().Str("comment", gzr.Comment).Msg("snapshot archive info")
		log.Info().Time("archive_time", gzr.ModTime).Msg("snapshot archive creation time")

		reader = gzr
	}

	// Create a decompressor with the default dictionary.
	decompressor, err := zstd.NewReader(reader, zstd.WithDecoderDicts(zbor.Dictionary))
	if err != nil {
		log.Error().Err(err).Msg("could not create decompressor")
		return failure
	}

	// Open the index database.
	db, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open badger db")
		return failure
	}
	defer db.Close()

	// Restore the database
	err = db.Load(decompressor, runtime.GOMAXPROCS(0))
	if err != nil {
		log.Error().Err(err).Msg("could not restore database")
		return failure
	}

	log.Info().Msg("snapshot restore complete")
	return success
}
