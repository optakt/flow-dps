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
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
)

// Forest represents a multitude of trees which are mapped by their state commitment hash.
type Forest interface {
	Save(tree *trie.MTrie, paths []ledger.Path, parent flow.StateCommitment)
	Has(commit flow.StateCommitment) bool
	Tree(commit flow.StateCommitment) (*trie.MTrie, bool)
	Paths(commit flow.StateCommitment) ([]ledger.Path, bool)
	Parent(commit flow.StateCommitment) (flow.StateCommitment, bool)
	Reset(finalized flow.StateCommitment)
}
