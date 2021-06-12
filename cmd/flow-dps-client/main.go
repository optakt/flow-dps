// Copyright 2021 Alvalor S.A.
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
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"

	"github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/rosetta/invoker"
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
		flagAPI    string
		flagHeight uint64
		flagLevel  string
		flagParams string
		flagScript string
	)

	pflag.StringVarP(&flagAPI, "api", "a", "", "host for GRPC API server")
	pflag.Uint64VarP(&flagHeight, "height", "h", 0, "block height to execute the script at")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagParams, "params", "p", "", "path to file with JSON-encoded list of Cadence arguments")
	pflag.StringVarP(&flagScript, "script", "s", "script.cdc", "path to file with Cadence script")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	// If no API server is given, choose based on height.
	if flagAPI == "" {
		for _, spork := range DefaultSporks {
			if flagHeight >= spork.First && flagHeight <= spork.Last {
				log.Info().Uint64("height", flagHeight).Str("spork", spork.Name).Str("api", spork.URL).Msg("spork and API chosen based on height")
				flagAPI = spork.URL
				break
			}
		}
	}
	if flagAPI == "" {
		log.Error().Uint64("height", flagHeight).Msg("could not find spork and API for height")
		return failure
	}

	// Initialize the API client.
	conn, err := grpc.Dial(flagAPI, grpc.WithInsecure())
	if err != nil {
		log.Error().Str("api", flagAPI).Err(err).Msg("could not dial API host")
		return failure
	}
	defer conn.Close()

	// Read the script.
	script, err := os.ReadFile(flagScript)
	if err != nil {
		log.Error().Str("script", flagScript).Err(err).Msg("could not read script")
		return failure
	}

	// Decode the arguments
	var args []cadence.Value
	if flagParams != "" {
		data, err := os.ReadFile(flagParams)
		if err != nil {
			log.Error().Err(err).Msg("could not read parameters")
			return failure
		}
		val, err := json.Decode(data)
		if err != nil {
			log.Error().Err(err).Msg("could not decode parameters")
			return failure
		}
		array, ok := val.(cadence.Array)
		if !ok {
			log.Error().Str("type", fmt.Sprintf("%T", val)).Msg("invalid type for parameters")
			return failure
		}
		args = array.Values
	}

	// Execute the script using remote lookup and read.
	client := dps.NewAPIClient(conn)
	invoke := invoker.New(dps.IndexFromAPI(client))
	result, err := invoke.Script(flagHeight, script, args)
	if err != nil {
		log.Error().Err(err).Msg("could not invoke script")
		return failure
	}
	output, err := json.Encode(result)
	if err != nil {
		log.Error().Uint64("height", flagHeight).Err(err).Msg("could not encode result")
		return failure
	}

	fmt.Println(string(output))

	return success
}
