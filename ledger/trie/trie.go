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
	"fmt"

	"github.com/gammazero/deque"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/hash"

	"github.com/optakt/flow-dps/models/dps"
)

// Trie is a modified Patricia-Merkle Trie, which is the storage layer of the Flow ledger.
// It uses a payload store to retrieve and persist ledger payloads.
type Trie struct {
	log   zerolog.Logger
	root  Node
	store dps.Store
}

// NewEmptyTrie creates a new trie without a root node, with the given payload store.
func NewEmptyTrie(log zerolog.Logger, store dps.Store) *Trie {
	t := Trie{
		log:   log.With().Str("subcomponent", "trie").Logger(),
		root:  nil,
		store: store,
	}

	return &t
}

// NewTrie creates a new trie using the given root node and payload store.
func NewTrie(log zerolog.Logger, root Node, store dps.Store) *Trie {
	t := Trie{
		log:   log.With().Str("subcomponent", "trie").Logger(),
		root:  root,
		store: store,
	}

	return &t
}

// RootNode returns the root node of the trie.
func (t *Trie) RootNode() Node {
	return t.root
}

func (t *Trie) Store() dps.Store {
	return t.store
}

// RootHash returns the hash of the trie's root node.
func (t *Trie) RootHash() ledger.RootHash {
	if t.root == nil {
		return ledger.RootHash(ledger.GetDefaultHashForHeight(ledger.NodeMaxHeight))
	}

	return ledger.RootHash(t.root.Hash())
}

// TODO: Add method to add multiple paths and payloads at once and parallelize insertions that do not conflict.
//  See https://github.com/optakt/flow-dps/issues/517

// Insert adds a new leaf to the trie. While doing so, since the trie is optimized, it might
// restructure the trie by adding new extensions, branches, or even moving other nodes
// to different heights along the way.
func (t *Trie) Insert(path ledger.Path, payload *ledger.Payload) {

	current := &t.root
	depth := uint16(0)
	for {
		switch node := (*current).(type) {
		case *Branch:
			// Since the new leaf is on this path, this branch is now dirty and needs its hash recomputed if it
			// was already computed.
			node.FlagDirty()

			// If the key bit at the index i is a 0, move on to the left child,
			// otherwise the right child.
			if bitutils.Bit(path[:], int(depth)) == 0 {
				current = &node.lChild
			} else {
				current = &node.rChild
			}
			depth++

		case *Extension:
			// Since the new leaf is on this path, this extension is now dirty and needs its hash recomputed if it
			// was already computed.
			node.FlagDirty()

			matched := commonBits(node.path, path)

			if matched == nodeHeight(node.skip)-1 {
				// The new leaf needs to be inserted precisely at one layer before the height up to which the
				// extension currently skips.
				// This is a special case to avoid creating an extension node that skips nothing and to use a branch
				// instead when a new leaf's path matches with all but the last bit of the extension's skipped path.

				// Create a branch to hold the original extension node's children.
				newBranch := NewBranch(nodeHeight(matched+1), node.lChild, node.rChild)

				// Create a new leaf to be the new child of the extension, along with the aforementioned branch.
				newLeaf := NewLeaf(nodeHeight(matched+1), path, payload)
				t.store.Save(newLeaf.Hash(), payload)

				var lChild, rChild Node
				if bitutils.Bit(path[:], int(matched)) == 0 {
					lChild = newLeaf
					rChild = newBranch
				} else {
					lChild = newBranch
					rChild = newLeaf
				}

				if node.height-node.skip == 1 {
					// Node only skipped one depth, so instead of moving it down under the new branch, it also needs
					// to be replaced with a branch.
					*current = NewBranch(nodeHeight(matched), lChild, rChild)
				} else {
					if node.height == node.skip+1 {
						fmt.Println("BUGGGGG")
					}
					// The node skipped more than one depth, so we simply shorten its skip value by one.
					*current = NewExtension(node.height, node.skip+1, node.path, lChild, rChild)
				}

				return
			}

			if matched == nodeHeight(node.height) {
				// The new leaf needs to be inserted precisely at the height below the extension's, and the
				// extension needs to be replaced with a branch.

				// Create new extension which starts lower but skips to the original height and path.
				if nodeHeight(matched+1) == node.skip {
					fmt.Println("BUGGGGG")
				}
				newExt := NewExtension(nodeHeight(matched+1), node.skip, node.path, node.lChild, node.rChild)

				// Set the children based on whether the new extension is needed on the left or right child.
				newLeaf := NewLeaf(nodeHeight(matched+1), path, payload)
				t.store.Save(newLeaf.Hash(), payload)

				var lChild, rChild Node
				if bitutils.Bit(path[:], int(matched)) == 0 {
					lChild = newLeaf
					rChild = newExt
				} else {
					lChild = newExt
					rChild = newLeaf
				}

				// Create new branch to replace current node.
				*current = NewBranch(node.height, lChild, rChild)
				return
			}

			if matched < nodeHeight(node.skip) {
				// The extension node is skipping over a path that is needed by the new leaf.
				// It needs to be shortened and a new extension node is needed at the intersection
				// of both paths.

				// Create new extension which starts lower but skips to the original height and path.
				if nodeHeight(matched+1) == node.skip {
					fmt.Println("BUGGGGG")
				}
				newExt := NewExtension(nodeHeight(matched+1), node.skip, node.path, node.lChild, node.rChild)

				// Set the children based on whether the new extension is needed on the left or right child.
				newLeaf := NewLeaf(nodeHeight(matched+1), path, payload)
				t.store.Save(newLeaf.Hash(), payload)

				var lChild, rChild Node
				if bitutils.Bit(path[:], int(matched)) == 0 {
					lChild = newLeaf
					rChild = newExt
				} else {
					lChild = newExt
					rChild = newLeaf
				}

				// Change children, path and skipped height of the original extension by recreating it.
				if node.height == nodeHeight(matched) {
					fmt.Println("BUGGGGG")
				}
				*current = NewExtension(node.height, nodeHeight(matched), path, lChild, rChild)
				return
			}

			// The path of the new leaf to insert matches with the skipped part of the extension's path, so we just
			// treat it like a branch and skip through many depths at once.
			if bitutils.Bit(path[:], int(nodeHeight(node.skip))) == 0 {
				current = &node.lChild
			} else {
				current = &node.rChild
			}
			depth = nodeHeight(node.skip - 1)

		case *Leaf:
			if node.path == path {
				// This path conflicts with a leaf, overwrite its hash using the new payload.
				node.hash = ledger.ComputeCompactValue(hash.Hash(path), payload.Value, int(node.height))
				t.store.Save(node.hash, payload)
				return
			}

			// This leaf is currently at a height which conflicts with the new path that we want to insert.
			// Therefore, we need to replace this leaf with a branch or extension that has two children, the
			// new leaf and the previous one.

			matched := commonBits(node.path, path)

			// We need to fetch the payload here since the old leaf now resides at a new height and therefore its
			// hash needs to be recomputed.
			oldPayload, err := t.store.Retrieve(node.Hash())
			if err != nil {
				t.log.Fatal().Err(err).Hex("path", node.path[:]).Msg("could not retrieve node payload from persistent storage")
			}

			oldLeaf := NewLeaf(nodeHeight(matched+1), node.path, oldPayload)
			t.store.Save(oldLeaf.Hash(), oldPayload)

			newLeaf := NewLeaf(nodeHeight(matched+1), path, payload)
			t.store.Save(newLeaf.Hash(), payload)

			// Compare first different bit between existing leaf and new leaf to know which one is which child for the
			// newly created branch.
			var lChild, rChild Node
			if bitutils.Bit(path[:], int(matched)) == 0 {
				lChild = newLeaf
				rChild = oldLeaf
			} else {
				lChild = oldLeaf
				rChild = newLeaf
			}

			// Create an extension node that skips up to the depth at which the old and
			// new leaves diverge in path, or a branch if the extension would not end up skipping anything.
			if depth == matched {
				// Create a branch, since an extension would not skip anything here.
				*current = NewBranch(nodeHeight(depth), lChild, rChild)
			} else {
				// Create an extension to skip over the common bits between both node paths.
				*current = NewExtension(nodeHeight(depth), nodeHeight(matched), path, lChild, rChild)
			}

			return

		case nil:
			// There is no leaf here yet, create it.
			*current = NewLeaf(nodeHeight(depth), path, payload)
			t.store.Save((*current).Hash(), payload)
			return
		}
	}
}

// Leaves iterates through the trie and returns its leaf nodes.
func (t *Trie) Leaves() []*Leaf {
	queue := deque.New()

	root := t.RootNode()
	if root != nil {
		queue.PushBack(root)
	}

	var leaves []*Leaf
	for queue.Len() > 0 {
		node := queue.PopBack().(Node)
		switch n := node.(type) {
		case *Leaf:
			leaves = append(leaves, n)
		case *Branch, *Extension:
			if node.LeftChild() != nil {
				queue.PushBack(node.LeftChild())
			}
			if node.RightChild() != nil {
				queue.PushBack(node.RightChild())
			}
		}
	}

	return leaves
}

// UnsafeRead read payloads for the given paths.
// CAUTION: If a given path is missing from the trie, this function panics.
func (t *Trie) UnsafeRead(paths []ledger.Path) []*ledger.Payload {
	payloads := make([]*ledger.Payload, len(paths)) // pre-allocate slice for the result
	for i := range paths {
		payloads[i] = t.read(paths[i])
	}
	return payloads
}

func (t *Trie) read(path ledger.Path) *ledger.Payload {
	current := &t.root
	depth := uint16(0)
	for {
		switch node := (*current).(type) {
		case *Branch:
			if bitutils.Bit(path[:], int(depth)) == 0 {
				current = &node.lChild
			} else {
				current = &node.rChild
			}
			depth++

		case *Extension:
			matched := commonBits(node.path, path)
			if matched < nodeHeight(node.skip) {
				// The path we are looking for is skipped in this trie, therefore it does not exist.
				return nil
			}

			if bitutils.Bit(path[:], int(nodeHeight(node.skip))) == 0 {
				current = &node.lChild
			} else {
				current = &node.rChild
			}
			depth = nodeHeight(node.skip - 1)

		case *Leaf:
			if node.path != path {
				// The path we are looking for is missing from this trie.
				return nil
			}

			payload, err := t.store.Retrieve(node.Hash())
			if err != nil {
				return nil
			}
			return payload

		case nil:
			return nil
		}
	}
}

// Converts depth into Flow Go inverted height (where 256 is root).
func nodeHeight(depth uint16) uint16 {
	return ledger.NodeMaxHeight - depth
}

// commonBits returns the number of matching bits within two paths.
func commonBits(path1, path2 ledger.Path) uint16 {
	for i := uint16(0); i < ledger.NodeMaxHeight; i++ {
		if bitutils.Bit(path1[:], int(i)) != bitutils.Bit(path2[:], int(i)) {
			return i
		}
	}

	return ledger.NodeMaxHeight
}
