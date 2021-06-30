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

	"github.com/onflow/flow-go/model/flow"
)

type State struct {
	forest Forest
	state  uint
	height uint64
	last   flow.StateCommitment
	next   flow.StateCommitment
	done   chan struct{}
}

func EmptyState(forest Forest) *State {
	s := &State{
		forest: forest,
		state:  stateEmpty,
		height: math.MaxUint64,
		last:   flow.StateCommitment{},
		next:   flow.StateCommitment{},
		done:   make(chan struct{}),
	}
	return s
}
