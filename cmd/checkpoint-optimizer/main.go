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

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/ledger/wal"
	"github.com/optakt/flow-dps/service/loader"
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
		flagInputPath  string
		flagLevel      string
		flagOutputPath string
	)

	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagInputPath, "input", "i", "./root.checkpoint", "path to the original checkpoint")
	pflag.StringVarP(&flagOutputPath, "output", "o", "./root.checkpoint", "path at which to write the optimized checkpoint")

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

	input, err := os.Open(flagInputPath)
	if err != nil {
		log.Error().Err(err).Msg("could not open input file")
		return failure
	}
	defer input.Close()

	output, err := os.Create(flagOutputPath)
	if err != nil {
		log.Error().Err(err).Msg("could not open output file")
		return failure
	}
	defer output.Close()

	original, err := loader.FromCheckpoint(log, input).Trie()
	if err != nil {
		log.Error().Err(err).Msg("could not read original checkpoint")
		return failure
	}

	paths, payloads := original.Values()

	optimized, err := trie.NewEmptyTrie().Mutate(paths, payloads)
	if err != nil {
		log.Error().Err(err).Msg("could not optimize checkpoint")
		return failure
	}

	err = wal.Checkpoint(output, optimized)
	if err != nil {
		log.Error().Err(err).Msg("could not save optimized checkpoint")
		return failure
	}

	return success
}
