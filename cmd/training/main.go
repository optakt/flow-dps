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
	"io/fs"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/awfm9/flow-dps/service/chain"
	"github.com/awfm9/flow-dps/service/feeder"
	"github.com/awfm9/flow-dps/service/mapper"
)

func main() {

	// Command line parameter initialization.
	var (
		flagLevel      string
		flagData       string
		flagTrie       string
		flagCheckpoint string
	)

	pflag.StringVarP(&flagLevel, "log-level", "l", "info", "log output level")
	pflag.StringVarP(&flagData, "data-dir", "d", "", "protocol state database directory")
	pflag.StringVarP(&flagTrie, "trie-dir", "t", "", "state trie write-ahead log directory")
	pflag.StringVarP(&flagCheckpoint, "checkpoint-file", "c", "", "state trie root checkpoint file")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// Initialize the mapper.
	chain, err := chain.FromProtocolState(flagData)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize chain")
	}
	feeder, err := feeder.FromLedgerWAL(flagTrie)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize feeder")
	}
	mapper, err := mapper.New(log, chain, feeder, &Index{}, mapper.WithCheckpointFile(flagCheckpoint))
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize mapper")
	}

	log.Info().Msg("running disk mapper to build trie")

	// Run the mapper to get the latest trie.
	start := time.Now()
	tree, err := mapper.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("disk mapper encountered error")
	}
	finish := time.Now()
	delta := finish.Sub(start)

	log.Info().Str("duration", delta.Round(delta).String()).Msg("disk mapper execution complete")

	// Now, we got the full trie and we can write random payloads to disk until
	// we have enough data for the dictionary creator.
	codec, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize codec")
	}
	payloads := allPayloads(tree.RootNode())
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(payloads), func(i int, j int) {
		payloads[i], payloads[j] = payloads[j], payloads[i]
	})
	total := uint64(0)
	for index, payload := range payloads {
		log := log.With().Int("index", index).Logger()
		data, err := codec.Marshal(payload)
		if err != nil {
			log.Fatal().Err(err).Msg("could not encode payload")
		}
		name := fmt.Sprintf("payload-%7d", index)
		err = ioutil.WriteFile(name, data, fs.ModePerm)
		if err != nil {
			log.Fatal().Err(err).Msg("could not write file")
		}
		total += uint64(len(data))
		log.Info().Int("file", len(data)).Uint64("total", total).Msg("wrote training file")
		if total > 100*112640 {
			break
		}
	}

	os.Exit(0)
}
