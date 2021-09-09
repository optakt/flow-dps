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
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/state/protocol"
	"github.com/onflow/flow-go/state/protocol/inmem"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"
	"github.com/onflow/flow-go/storage/badger/procedure"
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

	// Extract all the entities needed to bootstrap the state.
	result, seal, err := snapshot.SealedResult()
	if err != nil {
		return fmt.Errorf("could not get root seal and result: %w", err)
	}
	segment, err := snapshot.SealingSegment()
	if err != nil {
		return fmt.Errorf("could not get root segment: %w", err)
	}
	head, err := snapshot.Head()
	if err != nil {
		return fmt.Errorf("could not get root head: %w", err)
	}
	tail := segment[0].Header
	qc, err := snapshot.QuorumCertificate()
	if err != nil {
		return fmt.Errorf("could not get root QC: %w", err)
	}
	epochs := snapshot.Epochs()
	status := flow.EpochStatus{}

	err = db.Update(func(tx *badger.Txn) error {

		var parentID flow.Identifier
		for _, block := range segment {
			header := block.Header
			payload := block.Payload
			blockID := header.ID()

			// 1.1) HEADER
			err := operation.InsertHeader(blockID, header)(tx)
			if err != nil {
				return fmt.Errorf("could not insert header (block: %x): %w", blockID, err)
			}

			// 1.2.1) PAYLOAD GUARANTEES
			for _, guarantee := range payload.Guarantees {
				err = operation.InsertGuarantee(guarantee.CollectionID, guarantee)(tx)
				if err != nil {
					return fmt.Errorf("could not insert guarantee (block: %x, collection: %x): %w", blockID, guarantee.CollectionID, err)
				}
			}

			// 1.2.2) PAYLOAD SEALS
			for _, seal := range payload.Seals {
				err = operation.InsertSeal(seal.ID(), seal)(tx)
				if err != nil {
					return fmt.Errorf("could not insert guarantee (block: %x, seal: %x): %w", blockID, seal.ID(), err)
				}
			}

			// 1.2.3) PAYLOAD RESULTS
			for _, result := range payload.Results {
				err = operation.InsertExecutionResult(result)(tx)
				if err != nil {
					return fmt.Errorf("could not insert result (block: %x, result: %x): %w", blockID, result.ID(), err)
				}
			}

			// 1.2.4) PAYLOAD RECEIPTS
			for _, receipt := range payload.Receipts {
				err = operation.InsertExecutionReceiptMeta(receipt.ID(), receipt)(tx)
				if err != nil {
					return fmt.Errorf("could not insert receipt (block: %x, result: %x): %w", blockID, receipt.ResultID, err)
				}
				err = operation.IndexExecutionReceipts(blockID, receipt.ID())(tx)
				if err != nil {
					return fmt.Errorf("could not index receipt  (block: %x, result: %x): %w", blockID, receipt.ResultID, err)
				}
			}

			// 1.2.5) PAYLOAD INDEX
			err = procedure.InsertIndex(blockID, payload.Index())(tx)
			if err != nil {
				return fmt.Errorf("could not index payload (block: %x): %w", blockID, err)
			}

			// 1.3.1) VALIDITY
			err = operation.InsertBlockValidity(blockID, true)(tx)
			if err != nil {
				return fmt.Errorf("could not insert block validity (block: %x): %w", blockID, err)
			}

			// 1.3.2) HEIGHT
			err = operation.IndexBlockHeight(header.Height, blockID)(tx)
			if err != nil {
				return fmt.Errorf("could not index block height (block: %x): %w", blockID, err)
			}

			// 1.3.3) ANCESTRY
			err = operation.InsertBlockChildren(blockID, []flow.Identifier{})(tx)
			if err != nil {
				return fmt.Errorf("could not initialize ancestry (block: %x): %w", blockID, err)
			}
			err = operation.UpdateBlockChildren(parentID, []flow.Identifier{blockID})(tx)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				return fmt.Errorf("could not update ancestroy (block: %x, parent: %x): %w", blockID, parentID, err)
			}
			parentID = blockID
		}

		// 2.1) RESULT
		err = operation.InsertExecutionResult(result)(tx)
		if err != nil {
			return fmt.Errorf("could not insert root result: %w", err)
		}
		err = operation.IndexExecutionResult(result.BlockID, result.ID())(tx)
		if err != nil {
			return fmt.Errorf("could not index root result: %w", err)
		} // insert next epoch, if it exists

		// 2.2) SEAL
		err = operation.InsertSeal(seal.ID(), seal)(tx)
		if err != nil {
			return fmt.Errorf("could not insert root seal: %w", err)
		}
		err = operation.IndexBlockSeal(head.ID(), seal.ID())(tx)
		if err != nil {
			return fmt.Errorf("could not index root seal: %w", err)
		}

		// 3) QUORUM CERTIFICATE
		err = operation.InsertRootQuorumCertificate(qc)(tx)
		if err != nil {
			return fmt.Errorf("could not insert QC: %w", err)
		}

		// 4.1) VIEWS
		err = operation.InsertStartedView(head.ChainID, head.View)(tx)
		if err != nil {
			return fmt.Errorf("could not insert started view: %w", err)
		}
		err = operation.InsertVotedView(head.ChainID, head.View)(tx)
		if err != nil {
			return fmt.Errorf("could not insert voted view: %w", err)
		}

		// 4.2) HEIGHTS
		err = operation.InsertRootHeight(head.Height)(tx)
		if err != nil {
			return fmt.Errorf("could not insert root height: %w", err)
		}
		err = operation.InsertFinalizedHeight(head.Height)(tx)
		if err != nil {
			return fmt.Errorf("could not insert finalized height: %w", err)
		}
		err = operation.InsertSealedHeight(tail.Height)(tx)
		if err != nil {
			return fmt.Errorf("could not insert sealed height: %w", err)
		}

		// 5.1) EPOCH PREVIOUS
		previous := epochs.Previous()
		_, err := previous.Counter()
		if err != nil && !errors.Is(err, protocol.ErrNoPreviousEpoch) {
			return fmt.Errorf("could not get previous epoch counter: %w", err)
		}
		if err == nil {
			setup, err := protocol.ToEpochSetup(previous)
			if err != nil {
				return fmt.Errorf("could not get previous setup: %w", err)
			}
			err = operation.InsertEpochSetup(setup.ID(), setup)(tx)
			if err != nil {
				return fmt.Errorf("could not insert previous setup: %w", err)
			}
			commit, err := protocol.ToEpochCommit(previous)
			if err != nil {
				return fmt.Errorf("could not get previous commit: %w", err)
			}
			err = operation.InsertEpochCommit(commit.ID(), commit)(tx)
			if err != nil {
				return fmt.Errorf("could not insert previous commit: %w", err)
			}
			status.PreviousEpoch.SetupID = setup.ID()
			status.PreviousEpoch.CommitID = commit.ID()
		}

		// 5.2) EPOCH CURRENT
		current := epochs.Current()
		setup, err := protocol.ToEpochSetup(current)
		if err != nil {
			return fmt.Errorf("could not get current setup: %w", err)
		}
		err = operation.InsertEpochSetup(setup.ID(), setup)(tx)
		if err != nil {
			return fmt.Errorf("could not insert current setup: %w", err)
		}
		commit, err := protocol.ToEpochCommit(current)
		if err != nil {
			return fmt.Errorf("could not get current commit: %w", err)
		}
		err = operation.InsertEpochCommit(commit.ID(), commit)(tx)
		if err != nil {
			return fmt.Errorf("could not insert current commit: %w", err)
		}
		status.CurrentEpoch.SetupID = setup.ID()
		status.CurrentEpoch.CommitID = commit.ID()

		// 5.3) EPOCH NEXT
		next := epochs.Next()
		_, err = next.Counter()
		if err != nil && !errors.Is(err, protocol.ErrNextEpochNotSetup) {
			return fmt.Errorf("could not get next epoch counter: %w", err)
		}
		if err == nil {
			setup, err := protocol.ToEpochSetup(next)
			if err != nil {
				return fmt.Errorf("could not get next setup: %w", err)
			}
			err = operation.InsertEpochSetup(setup.ID(), setup)(tx)
			if err != nil {
				return fmt.Errorf("could not insert next setup: %w", err)
			}
			commit, err := protocol.ToEpochCommit(next)
			if err != nil {
				return fmt.Errorf("could not get next commit: %w", err)
			}
			err = operation.InsertEpochCommit(commit.ID(), commit)(tx)
			if err != nil {
				return fmt.Errorf("could not insert next commit: %w", err)
			}
			status.NextEpoch.SetupID = setup.ID()
			status.NextEpoch.CommitID = commit.ID()
		}

		// 5.4) EPOCH STATUS
		for _, block := range segment {
			blockID := block.Header.ID()
			err = operation.InsertEpochStatus(blockID, &status)(tx)
			if err != nil {
				return fmt.Errorf("could not insert epoch status (block: %x): %w", blockID, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not insert data: %w", err)
	}

	return nil
}
