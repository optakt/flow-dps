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

package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/bootstrap"
	"github.com/onflow/flow-go/module/metrics"
	protocol "github.com/onflow/flow-go/state/protocol/badger"
	"github.com/onflow/flow-go/state/protocol/inmem"
	"github.com/onflow/flow-go/storage"
	cache "github.com/onflow/flow-go/storage/badger"
	"github.com/onflow/flow-go/storage/badger/operation"
	"github.com/onflow/flow-go/utils/io"
)

// Initialize will initialize the Flow protocol state in the given database. The
// code is inspired by the related unexported code in the Flow Go code base:
// https://github.com/onflow/flow-go/blob/v0.21.0/cmd/bootstrap/cmd/finalize.go#L452
func Initialize(dir string, db *badger.DB) error {

	// Check if there is already a protocol state, in which case we error.
	var root uint64
	err := db.View(operation.RetrieveRootHeight(&root))
	if err == nil {
		return fmt.Errorf("protocol state already populated, please delete")
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("could not check root height: %w", err)
	}

	// Load the protocol snapshot from disk.
	data, err := io.ReadFile(filepath.Join(dir, bootstrap.PathRootProtocolStateSnapshot))
	if err != nil {
		return fmt.Errorf("could not read protocol snapshot file: %w", err)
	}
	var entities inmem.EncodableSnapshot
	err = json.Unmarshal(data, &entities)
	if err != nil {
		return fmt.Errorf("could not decode protocol snapshot: %w", err)
	}
	snapshot := inmem.SnapshotFromEncodable(entities)

	// Initialize the protocol state with the snapshot.
	collector := metrics.NewNoopCollector()
	headers := cache.NewHeaders(collector, db)
	index := cache.NewIndex(collector, db)
	guarantees := cache.NewGuarantees(collector, db, 0)
	seals := cache.NewSeals(collector, db)
	results := cache.NewExecutionResults(collector, db)
	receipts := cache.NewExecutionReceipts(collector, db, results, 0)
	payloads := cache.NewPayloads(db, index, guarantees, seals, receipts, results)
	_, err = protocol.Bootstrap(
		collector,
		db,
		cache.NewHeaders(collector, db),
		cache.NewSeals(collector, db),
		cache.NewExecutionResults(collector, db),
		cache.NewBlocks(db, headers, payloads),
		cache.NewEpochSetups(collector, db),
		cache.NewEpochCommits(collector, db),
		cache.NewEpochStatuses(collector, db),
		snapshot,
	)
	if err != nil {
		return fmt.Errorf("could not bootstrap protocol state: %w", err)
	}

	return nil
}
