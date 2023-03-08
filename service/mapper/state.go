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

package mapper

import (
	"math"

	"github.com/onflow/flow-go/ledger"
)

// State is the state machine's state for the current block being processed
type State struct {
	forest    Forest
	status    Status
	height    uint64
	updates   []*ledger.TrieUpdate
	registers map[ledger.Path]*ledger.Payload
	done      chan struct{}
}

// EmptyState returns a new empty state that uses the given forest.
func EmptyState(forest Forest) *State {

	s := State{
		forest:    forest,
		status:    StatusInitialize,
		height:    math.MaxUint64,
		registers: make(map[ledger.Path]*ledger.Payload),
		updates:   make([]*ledger.TrieUpdate, 0),
		done:      make(chan struct{}),
	}

	return &s
}
