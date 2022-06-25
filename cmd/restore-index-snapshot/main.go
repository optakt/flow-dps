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
	"runtime"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/service/index"
	"github.com/onflow/flow-dps/service/storage"
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
	)

	pflag.StringVarP(&flagCompression, "compression", "c", compressionZstd, "compression algorithm (\"none\", \"zstd\" or \"gzip\")")
	pflag.StringVarP(&flagEncoding, "encoding", "e", encodingNone, "output encoding (\"none\", \"hex\" or \"base64\")")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")

	pflag.Parse()

	// Initialize the logger.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)

	// Open the index database.
	db, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open badger db")
		return failure
	}
	defer db.Close()

	// Check if the database is empty.
	index := index.NewReader(db, storage.New(zbor.NewCodec()))
	_, err = index.First()
	if err == nil {
		log.Error().Msg("database directory already contains index database")
		return failure
	}

	// We will consume from stdin; if the user wants to load from a file, he can
	// pipe it into the command.
	var reader io.Reader
	reader = os.Stdin
	defer os.Stdin.Close()

	// When reading, we first need to decompress, so we start with that
	switch flagCompression {
	case compressionNone:
		// nothing to do
	case compressionZstd:
		decompressor, _ := zstd.NewReader(reader)
		defer decompressor.Close()
		reader = decompressor
	case compressionGzip:
		decompressor, _ := gzip.NewReader(reader)
		defer decompressor.Close()
		reader = decompressor
	default:
		log.Error().Str("compression", flagCompression).Msg("invalid compression algorithm specified")
	}

	// After decompression, we can decode the encoding.
	switch flagEncoding {
	case encodingNone:
		// nothing to do
	case encodingHex:
		reader = hex.NewDecoder(reader)
	case encodingBase64:
		reader = base64.NewDecoder(base64.StdEncoding, reader)
	default:
		log.Error().Str("encoding", flagEncoding).Msg("invalid encoding format specified")
	}

	// Restore the database
	err = db.Load(reader, runtime.GOMAXPROCS(0))
	if err != nil {
		log.Error().Err(err).Msg("snapshot restoration failed")
		return failure
	}

	log.Info().Msg("snapshot restoration complete")

	return success
}
