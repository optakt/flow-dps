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

	"github.com/optakt/flow-dps/models/dps"
)

func CatchupBlocks(db *badger.DB, read dps.Reader) ([]flow.Identifier, error) {

	indexed, err := read.Last()
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("could not get last indexed: %w", err)
	}
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

	var finalized uint64
	err = db.View(operation.RetrieveFinalizedHeight(&finalized))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get last finalized: %w", err)
	}

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
