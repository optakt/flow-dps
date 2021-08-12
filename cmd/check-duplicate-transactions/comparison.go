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

	"github.com/optakt/flow-dps/models/dps"
)

func compareDuplicates(log zerolog.Logger, dataDir string, indexDir string, duplicates map[uint64][]flow.Identifier) error {

	log.Info().Msg("comparing duplicates between databases")

	// Initialize the databases.
	protocol, err := badger.Open(dps.DefaultOptions(dataDir).WithReadOnly(true).WithBypassLockGuard(true))
	if err != nil {
		return fmt.Errorf("could not open protocol state (dir: %s): %w", dataDir, err)
	}
	defer protocol.Close()
	index, err := badger.Open(dps.DefaultOptions(indexDir).WithReadOnly(true).WithBypassLockGuard(true))
	if err != nil {
		return fmt.Errorf("could not open state index (dir: %s): %w", indexDir, err)
	}
	defer index.Close()

	log.Info().Msg("duplicate comparison completed")

	return nil
}
