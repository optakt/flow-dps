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

	"github.com/optakt/flow-dps/models/dps"
)

const (
	success = 0
	failure = 1

	updatesSamplePath      = "./samples/updates/"
	eventsSamplePath       = "./samples/events/"
	transactionsSamplePath = "./samples/transactions/"

	updatesDictionaryPath      = "./codec/zbor/updates"
	eventsDictionaryPath       = "./codec/zbor/events"
	transactionsDictionaryPath = "./codec/zbor/transactions"
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
		flagIndex string
		flagLevel string
	)

	pflag.StringVarP(&flagIndex, "index", "i", "index", "path to database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")

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

	// FIXME: Use the index database as the sample instead of the ledger WAL entries.
	//        Generate a random path and use the seek to select the next value after this path,
	//        and from this path take the last available height.

	// Initialize the index core state and open database in read-only mode.
	db, err := badger.Open(dps.DefaultOptions(flagIndex).WithReadOnly(true))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open index DB")
		return failure
	}
	defer db.Close()

	// FIXME: Delete samples before if they already exist.

	var duration time.Duration
	currentRatio := float64(0)
	previousRatio := float64(1)
	for size := 512; currentRatio < previousRatio*0.90; size = size * 2 {

		// Set previous ratio, except on first loop. FIXME: do this cleaner.
		if currentRatio != 0 {
			previousRatio = currentRatio
		}
		log.Info().Int("size", size).Float64("previous_ratio", previousRatio).Msg("generating payload dictionary")

		err = generatePayloadSamples(db, size*100)
		if err != nil {
			log.Error().Int("size", size).Err(err).Msg("could not generate payload samples")
			return failure
		}

		err = trainPayloadDictionary(size)
		if err != nil {
			log.Error().Err(err).Msg("could not generate dictionary for payloads")
			return failure
		}

		currentRatio, duration, err = benchmarkPayloadDictionary(db)
		if err != nil {
			log.Error().Err(err).Msg("could not benchmark dictionary for payloads")
			return failure
		}

		log.Info().
			Int("size", size).
			Float64("compression_ratio", currentRatio).
			Dur("compression_speed", duration).
			Msg("generated payload dictionary")
	}
	log.Info().
		Float64("previous_ratio", previousRatio).
		Float64("ratio", currentRatio).
		Dur("compression_speed", duration).
		Msg("done, stopped because ratio is good enough")

	// FIXME: For events select one type at a height and the sample is a list of events not one single event.

	//train = exec.Command("zstd", "--train", eventsSamplePath, "-o", eventsDictionaryPath)
	//err = train.Run()
	//if err != nil {
	//	panic(err)
	//}
	//
	//train = exec.Command("zstd", "--train", transactionsSamplePath, "-o", transactionsDictionaryPath)
	//err = train.Run()
	//if err != nil {
	//	panic(err)
	//}

	// FIXME: Use go templates to transform the dictionaries into proper Go files.

	return success
}
