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
		flagIndex string
		flagLevel string
		flagInput string
	)

	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVar(&flagInput, "input", "", "snapshot archive path")

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

	// Verify the snapshot archive path.
	if flagInput == "" {
		log.Error().Msg("snapshot archive path missing")
		return failure
	}

	// Open the input gzip file.
	file, err := os.Open(flagInput)
	if err != nil {
		log.Error().Str("file", flagInput).Err(err).Msg("could not open snapshot archive")
		return failure
	}
	defer file.Close()

	// Create a gzip reader.
	reader, err := gzip.NewReader(file)
	if err != nil {
		log.Error().Err(err).Msg("could not create gzip reader")
		return failure
	}

	log.Info().Str("file", flagInput).Msg("snapshot archive open ok")

	// Log the snapshot metadata.
	log.Info().Str("comment", reader.Comment).Msg("snapshot archive info")
	log.Info().Time("archive_time", reader.ModTime).Msg("snapshot archive creation time")

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
