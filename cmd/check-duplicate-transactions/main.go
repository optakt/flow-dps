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
	"errors"
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
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
		flagLevel string
	)

	pflag.StringVarP(&flagData, "data", "d", "data", "database directory for protocol state")
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

	// Open the index database.
	db, err := badger.Open(dps.DefaultOptions(flagData).WithReadOnly(true).WithBypassLockGuard(true))
	if err != nil {
		log.Error().Str("data", flagData).Err(err).Msg("could not open protocol state")
		return failure
	}
	defer db.Close()

	// Retrieve the root height as a start height for duplicate check.
	var root uint64
	err = db.View(operation.RetrieveRootHeight(&root))
	if err != nil {
		log.Error().Err(err).Msg("could not retrieve root height")
		return failure
	}

	// Go through each height, retrieve the transactions from the DB and check for duplicates.
	for height := root; ; height++ {
		txIDs := make(map[flow.Identifier]flow.Identifier)

		log := log.With().Uint64("height", height).Logger()

		// height => blockID
		var blockID flow.Identifier
		err = db.View(operation.LookupBlockHeight(uint64(height), &blockID))
		if errors.Is(err, storage.ErrNotFound) {
			log.Info().Msg("duplicate transaction check finished")
			break
		}
		if err != nil {
			log.Error().Err(err).Msg("could not look up height")
			return failure
		}

		log = log.With().Hex("block", blockID[:]).Logger()

		// blockID => collIDs
		var collIDs []flow.Identifier
		err = db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
		if err != nil {
			log.Error().Err(err).Msg("could not look up payload guarantees")
			return failure
		}

		for _, collID := range collIDs {

			log := log.With().Hex("collection", collID[:]).Logger()

			// collID => txIDs
			var collection flow.LightCollection
			err := db.View(operation.RetrieveCollection(collID, &collection))
			if err != nil {
				log.Error().Msg("could not retrieve collection")
				return failure
			}

			// txID ? duplicate
			for _, txID := range collection.Transactions {

				log := log.With().Hex("transaction", txID[:]).Logger()

				altID, ok := txIDs[txID]
				if ok {
					log.Info().Hex("alternative", altID[:]).Msg("transaction duplicated!")
					continue
				}

				txIDs[txID] = collID
				log.Debug().Msg("transaction not duplicated")
			}
		}
	}

	return success
}
