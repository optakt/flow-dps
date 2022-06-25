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
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/service/storage"
)

func indexCheck(log zerolog.Logger, dir string) (map[uint64][]flow.Identifier, error) {

	// We keep track of the duplicate transactions per height.
	duplicates := make(map[uint64][]flow.Identifier)

	log.Info().Str("index", dir).Msg("starting index state duplicate check")

	// Open the index database.
	index, err := badger.Open(dps.DefaultOptions(dir).WithReadOnly(true))
	if err != nil {
		return nil, fmt.Errorf("could not open state index (dir: %s): %w", dir, err)
	}
	defer index.Close()

	// Initialize the storage library.
	lib := storage.New(zbor.NewCodec())

	// Retrieve the root height as a start height for duplicate check.
	var first uint64
	err = index.View(lib.RetrieveFirst(&first))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve first: %w", err)
	}

	// Go through each height, retrieve the transactions from the DB and check for duplicates.
	for height := first; ; height++ {
		seen := make(map[flow.Identifier]struct{})

		log := log.With().Uint64("height", height).Logger()

		// height => txIDs
		var txIDs []flow.Identifier
		err = index.View(lib.LookupTransactionsForHeight(height, &txIDs))
		if errors.Is(err, badger.ErrKeyNotFound) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not look up transactions (height: %d): %w", height, err)
		}

		// txID ? duplicate
		var duplicateIDs []flow.Identifier
		for _, txID := range txIDs {

			log := log.With().Hex("transaction", txID[:]).Logger()

			_, ok := seen[txID]
			if ok {
				duplicateIDs = append(duplicateIDs, txID)
				log.Info().Msg("transaction duplicated!")
				continue
			}

			seen[txID] = struct{}{}
			log.Debug().Msg("transaction not duplicated")
		}

		// keep track of the duplicates at this height
		if len(duplicateIDs) > 0 {
			duplicates[height] = duplicateIDs
		}
	}

	log.Info().Msg("index state duplicate check complete")

	return duplicates, nil
}
