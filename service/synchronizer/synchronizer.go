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

package synchronizer

import (
	"errors"

	"github.com/onflow/flow-go/ledger"
)

type Synchronizer struct {
	rootHash    *ledger.RootHash
	blockHeight *uint64
}

func New() *Synchronizer {
	// FIXME: What happens if we already had an index file and we're not rebuilding it from scratch?
	return &Synchronizer{}
}

func (s *Synchronizer) SetRootHash(h ledger.RootHash) {
	s.rootHash = &h
}

func (s *Synchronizer) GetRootHash() (ledger.RootHash, error) {
	if s.rootHash == nil {
		return ledger.RootHash{}, errors.New("no root hash set yet")
	}

	return *s.rootHash, nil
}

func (s *Synchronizer) SetBlockHeight(height uint64) {
	s.blockHeight = &height
}

func (s *Synchronizer) GetBlockHeight() (uint64, error) {
	if s.blockHeight == nil {
		return 0, errors.New("no root hash set yet")
	}

	return *s.blockHeight, nil
}
