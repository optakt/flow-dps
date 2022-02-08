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
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/gammazero/deque"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/encoding"

	"github.com/optakt/flow-dps/models/dps"
)

const maxDepth = 255

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

// RootHash returns the hash of the trie's root node.
func (t *Trie) RootHash() ledger.RootHash {
	if t.root == nil {
		return ledger.RootHash(ledger.GetDefaultHashForHeight(ledger.NodeMaxHeight))
	}

	return t.root.Hash(ledger.NodeMaxHeight-1, [32]byte{}, t.store.Retrieve)
}

// TODO: Add method to add multiple paths and payloads at once and parallelize insertions that do not conflict.
//  See https://github.com/optakt/flow-dps/issues/517

// Insert adds a new leaf to the trie. While doing so, since the trie is optimized, it might
// restructure the trie by adding new extensions, branches, or even moving other nodes
// to different heights along the way.
func (t *Trie) Insert(path ledger.Path, payload *ledger.Payload) error {

	current := &t.root
	depth := uint8(0)
	for {
		switch node := (*current).(type) {

		// There are two cases where we hit a `nil` node:
		// - before reaching max depth, in which case we should create an
		// extension node to lead the rest of the path until max depth; and
		// - when reaching max depth, in which case we should place an empty
		// leaf node, which can then be populated.
		case nil:

			// When we have reached maximum depth, we can simply put a leaf node
			// into this location, which will then be populated in the leaf case.
			if depth == maxDepth {
				*current = &Leaf{}
				continue
			}

			// If we have not reached maximum depth, we have reached a part of
			// the trie that is empty, and we can reach the leaf's insertion
			// path by inserting an extension node for the rest of the path.
			extension := Extension{
				hash:  [32]byte{},
				dirty: true,
				path:  path[:],
				count: maxDepth - depth,
				child: nil,
			}
			*current = &extension
			current = &(extension.child)
			depth = maxDepth
			continue

		// Most of the nodes in a sparse trie will initially be made up of
		// extension nodes. They skip a part of the path where there are no
		// branches in order to reduce the number of nodes we need to traverse.
		case *Extension:

			// At this point, we count the number of common bits, so we can
			// compare it against the number of available bits on the extension.
			common := uint8(0)
			for i := depth; i < depth+node.count; i++ {
				if bitutils.Bit(path[:], int(i)) != bitutils.Bit(node.path[:], int(i)) {
					break
				}
				common++
			}

			depth += common

			// If all the bits are common, we have a simple edge case,
			// where we can skip to the end of the extension.
			if common == node.count {
				node.dirty = true
				current = &node.child
				continue
			}

			// Otherwise, we always have to create a fork in the path to our
			// leaf; one of the sides will remain `nil`, which is where we will
			// continue our traversal. The other side will contain whatever is
			// left of the extension node.
			branch := Branch{
				hash:  [32]byte{},
				dirty: true,
			}

			// If we have all but one bit in common, we have the branch on the
			// last bit, so the correct child for the previous extension side
			// of the new branch will point to the previous child of the branch.
			// Otherwise, we need to create a new branch with the remainder of
			// the previous path.
			var other Node
			if node.count-common == 1 {
				other = node.child
			} else {
				other = &Extension{
					hash:  [32]byte{},
					dirty: true,
					path:  node.path,
					count: node.count - common - 1,
					child: node.child,
				}
			}

			// If we have no bits in common, the first bit of the extension
			// should be replaced with the branch node, and the extension will
			// be garbage-collected. Otherwise, the extension points to the
			// branch, with a reduced path length.
			if common == 0 {
				*current = &branch
			} else {
				node.child = &branch
				node.count = common
				node.dirty = true
			}

			// Finally, we just have to point the wrong side of the branch,
			// which we will not follow, back at the previously existing path.
			if bitutils.Bit(path[:], int(depth)) == 0 {
				branch.right = other
				current = &branch.left
			} else {
				branch.left = other
				current = &branch.right
			}

			// Since we append a branch here, the depth of the next iteration
			// needs to be increased by one.
			depth++
			continue

		// Once the trie fills up more, we will have a lot of branch nodes,
		// where there are nodes on both the left and the right side. We can
		// simply continue iteration to the correct side.
		case *Branch:

			// If the key bit at the index i is a 0, move on to the left child,
			// otherwise the right child.
			if bitutils.Bit(path[:], int(depth)) == 0 {
				current = &node.left
			} else {
				current = &node.right
			}
			node.dirty = true
			depth++
			continue

		// When we reach a leaf node, we store the payload value in storage
		// and insert the node hash and payload hash into the leaf.
		case *Leaf:

			node.dirty = true

			if node.hash != [32]byte{} {
				// FIXME: Can this even happen?
				// FIXME: If the path is the exact same, we shouldnt create a branch but instead rehash.
				// In this case, we are conflicting with a previous leaf,
				// so we need to create a branch for both leaves.
				branch := Branch{
					hash:  [32]byte{},
					dirty: true,
				}

				if bitutils.Bit(path[:], int(depth)) == 0 {
					branch.left = &Leaf{
						hash: node.hash,
					}
					branch.right = node
				} else {
					branch.left = node
					branch.right = &Leaf{
						hash: node.hash,
					}
				}
				*current = &branch
				return nil
			}

			data := encoding.EncodePayload(payload)
			node.payload = sha256.Sum256(data)
			err := t.store.Save(node.payload, data)
			if err != nil {
				return err
			}
			return nil
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
		case *Extension:
			queue.PushBack(n.child)
		case *Branch:
			queue.PushBack(n.left)
			queue.PushBack(n.right)
		}
	}

	return leaves
}

// UnsafeRead read payloads for the given paths.
// CAUTION: If a given path is missing from the trie, this function panics.
func (t *Trie) UnsafeRead(paths []ledger.Path) []*ledger.Payload {
	payloads := make([]*ledger.Payload, len(paths)) // pre-allocate slice for the result
	for i := range paths {
		payload, err := t.read(paths[i])
		if errors.Is(err, ErrPathNotFound) {
			payloads[i] = nil
			continue
		}
		if err != nil {
			panic(err)
		}
		payloads[i] = payload
	}
	return payloads
}

func (t *Trie) read(path ledger.Path) (*ledger.Payload, error) {
	current := &t.root
	depth := uint8(0)
	for {
		switch node := (*current).(type) {

		// If we hit a `nil` node, it means nothing exists on that path and we
		// should return a `nil` payload.
		case nil:
			return nil, ErrPathNotFound

		// If we hit an extension node, we have two cases:
		// - the extension path overlaps fully with ours, and we jump to its end; or
		// - the extension path mismatches ours, and there is no value for our path.
		case *Extension:

			common := uint8(0)
			for i := depth; i < depth+node.count; i++ {
				if bitutils.Bit(path[:], int(i)) != bitutils.Bit(node.path[:], int(i)) {
					break
				}
				common++
			}
			if common != node.count {
				return nil, ErrPathNotFound
			}

			current = &node.child
			depth += node.count
			continue

		// If we hit a branch node, we have to sides to it, so we just forward
		// by one and go to the correct side.
		case *Branch:

			if bitutils.Bit(path[:], int(depth)) == 0 {
				current = &node.left
			} else {
				current = &node.right
			}
			depth++
			continue

		// Finally, if we reach the leaf, we can retrieve the by its hash from
		// storage and return it.
		case *Leaf:

			data, err := t.store.Retrieve(path)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve payload data: %w", err)
			}

			payload, err := encoding.DecodePayload(data)
			if err != nil {
				return nil, fmt.Errorf("could not decode payload data: %w", err)
			}

			return payload, nil
		}
	}
}

func (t *Trie) Paths() []ledger.Path {
	queue := deque.New()
	root := t.RootNode()
	if root != nil {
		queue.PushBack(root)
	}

	var paths []ledger.Path
	for queue.Len() > 0 {
		node := queue.PopBack().(Node)
		switch n := node.(type) {
		case *Extension:
			// If the child of this extension is not a leaf, add it to the queue.
			switch c := n.child.(type) {
			case *Extension, *Branch:
				queue.PushBack(c)
				continue
			}

			// Otherwise, we can stop here and add the path in the extension to
			// the result slice.
			path, err := ledger.ToPath(n.path)
			if err != nil {
				// An extension with a leaf child should always have a full path.
				panic(err)
			}
			paths = append(paths, path)

		case *Branch:
			if n.left != nil {
				queue.PushBack(n.left)
			}
			if n.right != nil {
				queue.PushBack(n.right)
			}
		}
	}

	return paths
}

func (t *Trie) Clone() *Trie {
	newTrie := &Trie{log: t.log, store: t.store}

	queue := deque.New()
	root := t.RootNode()
	if root != nil {
		queue.PushBack(root)
	}

	var prev, newNode Node
	for queue.Len() > 0 {
		node := queue.PopBack().(Node)

		switch n := node.(type) {
		case *Extension:
			// Clone extension node and link it to its parent.
			newExt := &Extension{
				count: n.count,
				path:  n.path,
			}
			newNode = newExt
			linkParent(prev, newExt)

			// Add its child to the queue.
			queue.PushBack(newExt.child)

		case *Branch:
			// Clone branch node and link it to its parent.
			newBranch := &Branch{
				left:  n.left,
				right: n.right,
			}
			newNode = newBranch
			linkParent(prev, newBranch)

			// Add its children to the queue.
			if newBranch.left != nil {
				queue.PushBack(newBranch.left)
			}
			if newBranch.right != nil {
				queue.PushBack(newBranch.right)
			}

		case *Leaf:
			// Clone leaf node and link it to its parent.
			newLeaf := &Leaf{
				hash: n.hash,
			}
			linkParent(prev, newLeaf)
		}

		prev = newNode
	}

	return newTrie
}

func linkParent(parent, child Node) {
	if parent == nil {
		return
	}

	switch p := parent.(type) {
	case *Extension:
		p.child = child
	case *Branch:
		if p.left == child {
			p.left = child
		}
		if p.right == child {
			p.right = child
		}
	}
}
