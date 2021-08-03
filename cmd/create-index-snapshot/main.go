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
	"path/filepath"
	"strings"
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
		flagRaw    bool
		flagOutput string
	)

	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.BoolVarP(&flagRaw, "raw", "r", false, "use raw binary output instead of hexadecimal when using stdout")
	pflag.StringVar(&flagOutput, "output", "", "output archive for snapshot")

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
	db, err := badger.Open(dps.DefaultOptions(flagIndex).WithReadOnly(true).WithBypassLockGuard(true))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open badger db")
		return failure
	}
	defer db.Close()

	// Create output writer.
	var writer io.Writer

	// If we're backing up to a file, prepare for writing gzip archive.
	if flagOutput != "" {

		target, err := getArchivePath(flagOutput)
		if err != nil {
			log.Error().Err(err).Str("output", flagOutput).Msg("invalid output path")
			return failure
		}

		log.Info().Str("target_file", target).Msg("output archive path")

		// Create the .gz output archive.
		file, err := os.Create(target)
		if err != nil {
			log.Error().Err(err).Str("output", target).Msg("could not create snapshot archive")
			return failure
		}
		defer file.Close()

		// Create a gzip writer. Data is written without compression
		// since we'll take care of compression ourselves.
		gzw, err := gzip.NewWriterLevel(file, gzip.NoCompression)
		if err != nil {
			log.Error().Err(err).Msg("could not create gzip writer")
			return failure
		}
		defer gzw.Close()

		gzw.Comment = fmt.Sprintf("DPS Index snapshot created at %v", time.Now().UTC().Format(timeLayout))
		gzw.ModTime = time.Now().UTC()

		writer = gzw

	} else {

		// Write to stdout and wrap the writer into a hex encoder if output is set to be hexadecimal.
		writer = os.Stdout
		if !flagRaw {
			writer = hex.NewEncoder(writer)
		}
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

func getArchivePath(out string) (string, error) {

	// If we were gven the full path to the archive, use that
	if strings.HasSuffix(out, ".gz") {
		return out, nil
	}

	// If not a full path, the output path should be a directory.
	info, err := os.Stat(out)
	if err != nil {
		return "", fmt.Errorf("could not stat path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("output must be a directory or a full file path")
	}

	// Output file will be a timestamped file in the specified directory.
	fileName := fmt.Sprintf("flow-dps-snapshot-%v.gz", time.Now().Format(timeLayout))
	targetFile := filepath.Join(out, fileName)

	return targetFile, nil
}
