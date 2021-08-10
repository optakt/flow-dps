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
	"fmt"
	"io"
	"os"
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

	timeLayout = "02-01-2006-15-04"
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
	pflag.StringVarP(&flagFormat, "format", "f", "hex", "output format (hex, gzip or raw)")

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

	// Validate output format.
	switch flagFormat {
	case "hex", "gzip", "raw":
		// Valid output formats.
	default:
		log.Error().Str("format", flagFormat).Msg("invalid format specified")
		return failure
	}

	// Open the index database.
	db, err := badger.Open(dps.DefaultOptions(flagIndex).WithReadOnly(true).WithBypassLockGuard(true))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open badger db")
		return failure
	}
	defer db.Close()

	// Create output writer.
	var writer io.Writer
	writer = os.Stdout
	if flagFormat == "hex" {
		writer = hex.NewEncoder(writer)

	} else if flagFormat == "gzip" {

		// Create a gzip writer. Data is written without compression
		// since we'll take care of compression ourselves.
		gzw, err := gzip.NewWriterLevel(writer, gzip.NoCompression)
		if err != nil {
			log.Error().Err(err).Msg("could not create gzip writer")
			return failure
		}
		defer gzw.Close()

		gzw.Comment = fmt.Sprintf("DPS Index snapshot created at %v", time.Now().UTC().Format(timeLayout))
		gzw.ModTime = time.Now().UTC()

		writer = gzw
	}

	// Create a compressor to make sure only compressed bytes are piped into the writer.
	compressor, err := zstd.NewWriter(writer,
		zstd.WithEncoderDict(zbor.Dictionary),
	)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize compressor")
		return failure
	}
	defer compressor.Close()

	// Run the DB backup mechanism on top of the writer to directly
	// write the output.
	_, err = db.Backup(compressor, 0)
	if err != nil {
		log.Error().Err(err).Msg("could not backup database")
		return failure
	}

	return success
}
