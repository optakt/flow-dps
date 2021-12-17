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

package trie

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
)

// TODO: Look into arena allocation for node paths to improve both memory usage
//  and performance. See https://github.com/optakt/flow-dps/issues/518.
// TODO: Look into using a sync Pool to reduce allocations at the expense of
//  some performance. See https://github.com/optakt/flow-dps/issues/519.

// Node represents a trie node.
type Node interface {
	Height() uint16
	Path() ledger.Path
	Hash() hash.Hash

	LeftChild() Node
	RightChild() Node
}
