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
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/onflow/flow-dps/codec/generator"
	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/models/dps"
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
		flagDictionaryPath string
		flagIndex          string
		flagLevel          string
		flagSamplePath     string
		flagStartSize      int
		flagTolerance      float64
	)

	pflag.StringVar(&flagDictionaryPath, "dictionary-path", "./codec/zbor", "path to the package in which to write dictionaries")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "path to database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVar(&flagSamplePath, "sample-path", "", "path to the directory in which to store samples for dictionary training (temporary folder when left empty)")
	pflag.IntVar(&flagStartSize, "start-size", 512, "minimum dictionary size in bytes to generate (will be doubled on each iteration)")
	pflag.Float64Var(&flagTolerance, "tolerance", 0.1, "compression ratio increase tolerance, between 0 and 1")

	pflag.Parse()

	// Increase the GOMAXPROCS value in order to use the full IOPS available, see:
	// https://groups.google.com/g/golang-nuts/c/jPb_h3TvlKE
	_ = runtime.GOMAXPROCS(128)

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	// Initialize the index core state and open database in read-only mode.
	db, err := badger.Open(dps.DefaultOptions(flagIndex).WithReadOnly(true))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open index DB")
		return failure
	}
	defer db.Close()

	codec := zbor.NewCodec()

	samplePath := flagSamplePath
	if flagSamplePath == "" {
		samplePath, err = os.MkdirTemp("", "samples")
		if err != nil {
			log.Error().Str("sample_path", flagSamplePath).Err(err).Msg("could not create temporary directory")
			return failure
		}
	}
	defer os.RemoveAll(samplePath)

	generate := generator.New(log, db, codec,
		generator.WithDictionaryPath(flagDictionaryPath),
		generator.WithSamplePath(samplePath),
		generator.WithRatioImprovementTolerance(flagTolerance),
		generator.WithStartSize(flagStartSize),
	)

	err = generate.Dictionary(generator.KindPayloads)
	if err != nil {
		log.Error().Err(err).Msg("could not generate payload dictionary")
		return failure
	}

	err = generate.Dictionary(generator.KindTransactions)
	if err != nil {
		log.Error().Err(err).Msg("could not generate transactions dictionary")
		return failure
	}

	err = generate.Dictionary(generator.KindEvents)
	if err != nil {
		log.Error().Err(err).Msg("could not generate events dictionary")
		return failure
	}

	return success
}
