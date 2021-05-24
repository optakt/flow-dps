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
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/service/storage"
)

type Height struct {
	core *Core
}

func (h *Height) ForBlock(blockID flow.Identifier) (uint64, error) {
	var height uint64
	err := h.core.db.View(storage.RetrieveHeightByBlock(blockID, &height))
	return height, err
}

func (h *Height) ForCommit(commit flow.StateCommitment) (uint64, error) {
	var height uint64
	err := h.core.db.View(storage.RetrieveHeightByCommit(commit, &height))
	return height, err
}
