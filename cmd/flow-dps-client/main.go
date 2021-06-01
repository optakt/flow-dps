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
	"io/ioutil"
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
	"github.com/optakt/flow-dps/rosetta/lookup"
	"github.com/optakt/flow-dps/rosetta/read"
)

func main() {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagAPI    string
		flagHeight uint64
		flagLog    string
		flagParams string
		flagScript string
	)

	pflag.StringVarP(&flagAPI, "api", "a", "127.0.0.1:5005", "host for GRPC API server")
	pflag.Uint64VarP(&flagHeight, "height", "h", 0, "block height to execute the script at")
	pflag.StringVarP(&flagLog, "log", "l", "info", "log output level")
	pflag.StringVarP(&flagParams, "params", "p", "", "JSON encoded Cadence parameters for the script")
	pflag.StringVarP(&flagScript, "script", "s", "script.cdc", "path to the Cadence script file to be executed")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLog)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// Initialize the API client.
	conn, err := grpc.Dial(flagAPI, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Msg("could not dial API host")
	}
	client := dps.NewAPIClient(conn)

	// Read the script.
	script, err := ioutil.ReadFile(flagScript)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read script")
	}

	// Decode the arguments
	var args []cadence.Value
	if flagParams != "" {
		val, err := json.Decode([]byte(flagParams))
		if err != nil {
			log.Fatal().Err(err).Msg("could not decode parameters")
		}
		array, ok := val.(cadence.Array)
		if !ok {
			log.Fatal().Str("type", fmt.Sprintf("%T", val)).Msg("invalid type for parameters")
		}
		args = array.Values
	}

	// Execute the script using remote lookup and read.
	invoke := invoker.New(lookup.FromDPS(client), read.FromDPS(client))
	result, err := invoke.Script(flagHeight, script, args)
	if err != nil {
		log.Error().Err(err).Msg("could not execute script")
	}
	output, err := json.Encode(result)
	if err != nil {
		log.Fatal().Err(err).Msg("could not encode result")
	}

	fmt.Println(string(output))

	os.Exit(0)
}
