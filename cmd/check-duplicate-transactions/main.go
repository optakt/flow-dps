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
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/onflow/flow-go/model/flow"
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
		flagData  string
		flagIndex string
		flagLevel string
	)

	pflag.StringVarP(&flagData, "data", "d", "", "database directory for protocol state")
	pflag.StringVarP(&flagIndex, "index", "i", "", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")

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

	// We should have at least one of data or index directories.
	if flagData == "" && flagIndex == "" {
		log.Error().Msg("need at least one of data or index directories")
		return failure
	}

	// Only check the protocol state if a directory for it is given.
	if flagData != "" {
		err = protocolCheck(log, flagData)
		if err != nil {
			log.Error().Err(err).Msg("could not execute protocol state duplicate check")
			return failure
		}
	}

	// This keeps track of heights on the state index that have duplicate
	// transactions and checks against the protocol state where available.

	// Only check the state index if a directory for it is given.
	var duplicates map[uint64][]flow.Identifier
	if flagIndex != "" {
		duplicates, err = indexCheck(log, flagIndex)
		if err != nil {
			log.Error().Err(err).Msg("could not execute state index duplicate check")
			return failure
		}
	}

	// If we have both a protocol state and a state index database, we can check
	// duplicates from the state index against the protocol state.
	if flagData != "" && flagIndex != "" && len(duplicates) > 0 {
		err := compareDuplicates(log, flagData, flagIndex, duplicates)
		if err != nil {
			log.Error().Err(err).Msg("could not compare duplicates")
			return failure
		}
	}

	return success
}
