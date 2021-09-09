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
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/consensus/hotstuff/notifications/pubsub"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"
)

func Initialize(db *badger.DB, callback pubsub.OnBlockFinalizedConsumer) error {

	var root uint64
	err := db.View(operation.RetrieveRootHeight(&root))
	if err != nil {
		return fmt.Errorf("could not retrieve root height: %w", err)
	}

	for height := root + 1; ; height++ {

		var blockID flow.Identifier
		err = db.View(operation.LookupBlockHeight(height, &blockID))
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("could not look up block: %w", err)
		}

		callback(blockID)
	}
}
