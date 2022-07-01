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
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/service/storage"
)

func compareDuplicates(log zerolog.Logger, dataDir string, indexDir string, duplicates map[uint64][]flow.Identifier) error {

	log.Info().Msg("comparing duplicates between databases")

	// Initialize the databases.
	protocol, err := badger.Open(dps.DefaultOptions(dataDir).WithReadOnly(true))
	if err != nil {
		return fmt.Errorf("could not open protocol state (dir: %s): %w", dataDir, err)
	}
	defer protocol.Close()
	index, err := badger.Open(dps.DefaultOptions(indexDir).WithReadOnly(true))
	if err != nil {
		return fmt.Errorf("could not open state index (dir: %s): %w", indexDir, err)
	}
	defer index.Close()

	// Initialize the storage library.
	lib := storage.New(zbor.NewCodec())

	// Go through duplicates and compare number and IDs of duplicates beetween databases.
	for height, duplicateIDs := range duplicates {

		log := log.With().Uint64("height", height).Int("duplicates", len(duplicateIDs)).Logger()

		// create a lookup of duplicate IDs
		lookup := make(map[flow.Identifier]struct{})
		for _, duplicateID := range duplicateIDs {
			lookup[duplicateID] = struct{}{}
		}

		// height => blockID
		var blockID flow.Identifier
		err = protocol.View(operation.LookupBlockHeight(uint64(height), &blockID))
		if err != nil {
			return fmt.Errorf("could not look up block (height: %d): %w", height, err)
		}

		// blockID => collIDs
		var collIDs []flow.Identifier
		err = protocol.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
		if err != nil {
			return fmt.Errorf("could not look up payload guarantees (block: %x): %w", blockID, err)
		}

		// collIDs => txIDs
		var firstFound uint
		var firstCount uint
		for _, collID := range collIDs {

			log := log.With().Hex("collection", collID[:]).Logger()

			// collID => txIDs
			var collection flow.LightCollection
			err := protocol.View(operation.RetrieveCollection(collID, &collection))
			if err != nil {
				return fmt.Errorf("could not retrieve collection (%x): %w", collID, err)
			}

			// txID ? in duplicate IDs
			firstCount += uint(len(collection.Transactions))
			for _, txID := range collection.Transactions {
				_, ok := lookup[txID]
				if ok {
					firstFound++
					log.Debug().Hex("transaction", txID[:]).Msg("duplicate from protocol state found in duplicates")
				}
			}
		}

		var txIDs []flow.Identifier
		err = index.View(lib.LookupTransactionsForHeight(height, &txIDs))
		if err != nil {
			return fmt.Errorf("could not look up transactions (height: %d): %w", height, err)
		}

		// txID ? in duplicate IDs
		// height => txIDs
		secondCount := uint(len(txIDs))
		var secondFound uint
		for _, txID := range txIDs {
			_, ok := lookup[txID]
			if ok {
				secondFound++
				log.Debug().Hex("transaction", txID[:]).Msg("transaction from state index found in duplicates")
			}
		}

		log.Info().Uint64("height", height).Hex("block", blockID[:]).Uint("first_count", firstCount).Uint("second_count", secondCount).Uint("first_found", firstFound).Uint("second_found", secondFound).Msg("compared duplicates for block")
	}

	log.Info().Msg("duplicate comparison completed")

	return nil
}
