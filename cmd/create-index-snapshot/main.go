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
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

func main() {

	var (
		flagDir      string
		flagLogLevel string
		flagRaw      string
	)

	pflag.StringVarP(&flagDir, "dir", "d", "", "path to badger database")
	pflag.StringVarP(&flagLogLevel, "log", "l", "info", "log level for JSON logger")
	pflag.StringVarP(&flagRaw, "raw", "r", "", "target file for raw output (overwrites existing)")

	pflag.Parse()

	zerolog.TimestampFunc = func() time.Time { return time.Now() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Str("level", flagLevel).Err(err).Msg("could not parse log level")
	}

	log = log.Level(level)

	db, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Fatal().Str("index", flagIndex).Err(err).Msg("could not open badger db")
	}
	defer db.Close()

	var buf bytes.Buffer
	compressor, err := zstd.NewWriter(&buf)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize zstd compression")
	}
	defer compressor.Close()

	_, err = db.Backup(compressor, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("could not backup badger db")
	}

	err = compressor.Close()
	if err != nil {
		log.Warn().Err(err).Msg("could not close compression writer")
	}

	// if we don't want binary output, just write to stdout and we're done
	if flagRaw == "" {
		fmt.Printf("%s", hex.EncodeToString(buf.Bytes()))
		return
	}

	// open output file - create/truncate existing
	out, err := os.OpenFile(flagRaw, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal().Err(err).Str("path", flagRaw).Msg("could not open output file")
	}

	defer out.Close()

	_, err = out.Write(buf.Bytes())
	if err != nil {
		log.Fatal().Err(err).Str("path", flagRaw).Msg("could not write to output")
	}
}
