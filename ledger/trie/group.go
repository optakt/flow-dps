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
)

// Group represents a group of values that need to  be inserted into a target trie.
type Group struct {
	// The group's path, which is the left-most element of all the paths
	// that the group contains.
	path *ledger.Path

	// The source pointer points to a node that matches the same position as the
	// target pointer, in the source trie.
	source Pointer
	// The target pointer points to the node we are currently building in the
	// new trie.
	target Pointer
	// start is the start index of this group, in the paths slice that is
	// given to the trie.Mutate method.
	start uint
	// end is the end index of this group, in the paths slice that is given to
	// the trie.Mutate method.
	end uint
	// depth keeps track of the depth at which the group currently is, in the
	// trie.
	depth uint8
	// leaf represents whether we are at the end of the trie, where leaves should
	// be created.
	leaf bool
}

// Pointer is a structure that keeps track of a node pointer, and if that node is
// an extension, can be used to keep track of its position on the extension.
type Pointer struct {
	node  *Node
	count uint8
}
