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
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/optakt/flow-dps/models/dps"
)

const (
	success = 0
	failure = 1
)

const (
	encodingNone   = "none"
	encodingHex    = "hex"
	encodingBase64 = "base64"
)

const (
	compressionNone = "none"
	compressionZstd = "zstd"
	compressionGzip = "gzip"
)

func main() {
	os.Exit(run())
}

func run() int {

	// Parse the command line arguments.
	var (
		flagCompression string
		flagEncoding    string
		flagIndex       string
		flagLevel       string
	)

	pflag.StringVarP(&flagCompression, "compression", "c", "zstd", "compression algorithm (`none`, `zstd` or `gzip`)")
	pflag.StringVarP(&flagEncoding, "encoding", "e", "none", "output encoding (`none`, `hex` or `base64`)")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "severity level for logging output")

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

	// Open the index database.
	db, err := badger.Open(dps.DefaultOptions(flagIndex).WithReadOnly(true))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open badger db")
		return failure
	}
	defer db.Close()

	// Create the writer(s) for the output format.
	var writer io.Writer
	writer = os.Stdout
	switch flagEncoding {
	case encodingNone:
		// nothing to do
	case encodingHex:
		writer = hex.NewEncoder(writer)
	case encodingBase64:
		encoder := base64.NewEncoder(base64.StdEncoding, writer)
		defer encoder.Close()
		writer = encoder
	default:
		log.Error().Str("encoding", flagEncoding).Msg("invalid encoding specified")
	}

	// Wrap the output writer in a compressing writer of the given algorithm.
	switch flagCompression {
	case compressionNone:
		// nothing to do
	case compressionZstd:
		compressor, _ := zstd.NewWriter(writer, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
		defer compressor.Close()
		writer = compressor
	case compressionGzip:
		compressor, _ := gzip.NewWriterLevel(writer, gzip.BestCompression)
		defer compressor.Close()
		writer = compressor
	}

	// Run the DB backup mechanism on top of the writer to create the snapshot.
	_, err = db.Backup(writer, 0)
	if err != nil {
		log.Error().Err(err).Msg("could not backup database")
		return failure
	}

	return success
}
