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
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/onflow/flow-dps/models/dps"
)

func protocolCheck(log zerolog.Logger, dir string) error {

	log.Info().Str("data", dir).Msg("starting protocol state duplicate check")

	// Open the protocol state database.
	protocol, err := badger.Open(dps.DefaultOptions(dir).WithReadOnly(true))
	if err != nil {
		return fmt.Errorf("could not open protocol state (dir: %s): %w", dir, err)
	}
	defer protocol.Close()

	// Retrieve the root height as a start height for duplicate check.
	var root uint64
	err = protocol.View(operation.RetrieveRootHeight(&root))
	if err != nil {
		return fmt.Errorf("could not retrieve root height: %w", err)
	}

	// Go through each height, retrieve the transactions from the DB and check for duplicates.
	for height := root; ; height++ {
		seen := make(map[flow.Identifier]flow.Identifier)

		log := log.With().Uint64("height", height).Logger()

		// height => blockID
		var blockID flow.Identifier
		err = protocol.View(operation.LookupBlockHeight(uint64(height), &blockID))
		if errors.Is(err, storage.ErrNotFound) {
			break
		}
		if err != nil {
			return fmt.Errorf("could not look up block (height: %d): %w", height, err)
		}

		log = log.With().Hex("block", blockID[:]).Logger()

		// blockID => collIDs
		var collIDs []flow.Identifier
		err = protocol.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
		if err != nil {
			return fmt.Errorf("could not look up payload guarantees (block: %x): %w", blockID, err)
		}

		for _, collID := range collIDs {

			log := log.With().Hex("collection", collID[:]).Logger()

			// collID => txIDs
			var collection flow.LightCollection
			err := protocol.View(operation.RetrieveCollection(collID, &collection))
			if err != nil {
				return fmt.Errorf("could not retrieve collection (%x): %w", collID, err)
			}

			// txID ? duplicate
			for _, txID := range collection.Transactions {

				log := log.With().Hex("transaction", txID[:]).Logger()

				altID, ok := seen[txID]
				if ok {
					log.Info().Hex("alternative", altID[:]).Msg("transaction duplicated!")
					continue
				}

				seen[txID] = collID
				log.Debug().Msg("transaction not duplicated")
			}
		}
	}

	log.Info().Msg("protocol state duplicate check complete")

	return nil
}
