package initializer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/module/metrics"
	protocol "github.com/onflow/flow-go/state/protocol/badger"
	"github.com/onflow/flow-go/state/protocol/inmem"
	"github.com/onflow/flow-go/storage"
	cache "github.com/onflow/flow-go/storage/badger"
	"github.com/onflow/flow-go/storage/badger/operation"
)

// ProtocolState initializes the Flow protocol state in the given database. The
// code is inspired by the related unexported code in the Flow Go code base:
// https://github.com/onflow/flow-go/blob/v0.21.0/cmd/bootstrap/cmd/finalize.go#L452
// TODO(leo): use the code from flow-go
func ProtocolState(file io.Reader, db *badger.DB) error {

	// If we already have a root height, skip bootstrapping.
	var root uint64
	err := db.View(operation.RetrieveRootHeight(&root))
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("could not check root: %w", err)
	}
	if err == nil {
		return nil
	}

	// Load the protocol snapshot from disk.
	data, err := io.ReadAll(file)
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
	all := cache.InitAll(collector, db)

	_, err = protocol.Bootstrap(
		collector,
		db,
		all.Headers,
		all.Seals,
		all.Results,
		all.Blocks,
		all.QuorumCertificates,
		all.Setups,
		all.EpochCommits,
		all.Statuses,
		all.VersionBeacons,
		snapshot,
	)
	if err != nil {
		return fmt.Errorf("could not bootstrap protocol state: %w", err)
	}

	return nil
}
