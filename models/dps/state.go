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

package dps

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type State interface {
	Index() Index
	Chain() Chain
	Last() Last
	Height() Height
	Commit() Commit
	Raw() Raw
	Ledger() Ledger
}

type Index interface {
	Header(height uint64, header *flow.Header) error
	Commit(height uint64, commit flow.StateCommitment) error
	Deltas(height uint64, deltas []Delta) error
	Events(height uint64, events []flow.Event) error
	Last(commit flow.StateCommitment) error
	Compact() error
}

type Chain interface {
	Header(height uint64) (*flow.Header, error)
	Events(height uint64, types ...string) ([]flow.Event, error)
}

type Last interface {
	Height() uint64
	Commit() flow.StateCommitment
}

type Height interface {
	ForBlock(blockID flow.Identifier) (uint64, error)
	ForCommit(commit flow.StateCommitment) (uint64, error)
}

type Commit interface {
	ForHeight(height uint64) (flow.StateCommitment, error)
}

type Raw interface {
	WithHeight(height uint64) Raw
	Get(key []byte) ([]byte, error)
}

type Ledger interface {
	WithVersion(version uint8) Ledger
	Get(*ledger.Query) ([]ledger.Value, error)
}
