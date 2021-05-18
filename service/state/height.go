// Copyright 2021 Alvalor S.A.
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

package state

import (
	"fmt"

	"github.com/awfm9/flow-dps/service/state/storage"
	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"
)

type Height struct {
	core *Core
}

// TODO: move all core logic to the core state and just proxy to unexported
// functions from the sub-interfaces
// => https://github.com/awfm9/flow-dps/issues/37

func (h *Height) ForBlock(blockID flow.Identifier) (uint64, error) {
	var height uint64
	err := h.core.db.View(func(tx *badger.Txn) error {
		return storage.RetrieveHeightByBlock(blockID[:], &height)(tx)
	})
	if err != nil {
		return 0, fmt.Errorf("could not look up block: %w", err)
	}
	return height, nil
}

func (h *Height) ForCommit(commit flow.StateCommitment) (uint64, error) {
	var height uint64
	err := h.core.db.View(func(tx *badger.Txn) error {
		return storage.RetrieveHeightByCommit(commit, &height)(tx)
	})
	if err != nil {
		return 0, fmt.Errorf("could not look up commit: %w", err)
	}
	return height, nil
}
