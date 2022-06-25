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

package initializer

import (
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/onflow/flow-dps/models/dps"
)

// CatchupBlocks will determine, based on what is in the protocol state and
// index databases, which blocks we need to download the execution records for
// in order to properly resume catching up with consensus.
func CatchupBlocks(db *badger.DB, read dps.Reader) ([]flow.Identifier, error) {

	// We need to know for which blocks we don't need the execution records
	// anymore, which is basically up to the last indexed block.
	indexed, err := read.Last()
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("could not get last indexed: %w", err)
	}

	// If there is no last indexed block, we should start downloading execution
	// records just after root height (for all the blocks), so we put the
	// last indexed height at root. If there is no root height, we don't need
	// to catch up with anything, because the protocol state is also empty.
	if errors.Is(err, badger.ErrKeyNotFound) {
		var root uint64
		err = db.View(operation.RetrieveRootHeight(&root))
		if errors.Is(err, storage.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("could not get root height: %w", err)
		}
		indexed = root
	}

	// Next, we check what the last finalized height in the protocol state is.
	// If we can't find it, we don't have to catch up with anything; it's
	// possible we are starting from scratch, or that we removed the protocol
	// state without removing the index. In that case, we will simply re-index
	// everything as we are syncing the protocol state.
	var finalized uint64
	err = db.View(operation.RetrieveFinalizedHeight(&finalized))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get last finalized: %w", err)
	}

	// We can now step from the first height after the indexed height to the
	// finalized height and collect all the block IDs on the way. These can then
	// be queued in the cloud streamer to download the block records for blocks
	// that have not yet been indexed in the correct order.
	var blockIDs []flow.Identifier
	for height := indexed + 1; height <= finalized; height++ {
		var blockID flow.Identifier
		err = db.View(operation.LookupBlockHeight(height, &blockID))
		if err != nil {
			return nil, fmt.Errorf("could not look up block (height: %d): %w", height, err)
		}
		blockIDs = append(blockIDs, blockID)
	}

	return blockIDs, nil
}
