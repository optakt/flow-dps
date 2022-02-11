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
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/gammazero/deque"
	"github.com/rs/zerolog"
	"lukechampine.com/blake3"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/encoding"
	trie "github.com/onflow/flow-go/ledger/common/hash"

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

	return t.root.Hash(ledger.NodeMaxHeight - 1)
}

// TODO: Add method to add multiple paths and payloads at once and parallelize insertions that do not conflict.
//  See https://github.com/optakt/flow-dps/issues/517

// Insert adds a new leaf to the trie. While doing so, since the trie is optimized, it might
// restructure the trie by adding new extensions, branches, or even moving other nodes
// to different heights along the way.
func (t *Trie) Insert(path ledger.Path, payload *ledger.Payload) error {

	// Insertions should never fail, so we can start by encoding the payload
	// data and storing it in our key-value store. We can also check whether the
	// KV store already has this data stored, to avoid unnecessary I/O.
	data := encoding.EncodePayload(payload)
	key := blake3.Sum256(data)
	err := t.store.Has(key)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return fmt.Errorf("could not check payload data in store: %w", err)
	}
	if errors.Is(err, badger.ErrKeyNotFound) {
		err = t.store.Save(key, data)
	}
	if err != nil {
		return fmt.Errorf("could not save payload data to store: %w", err)
	}

	// Current always points at the current node for the iteration. It's the
	// pointer that we forward while iterating along the path of the insertion.
	// We can modify the trie while iterating by replacing its contents.
	// Previous keeps the previous
	var parent *Node
	current := &t.root

	// Depth keeps track of the depth that we are at in the trie. The root node
	// is at a depth of zero; every branch node adds a depth of one, while every
	// extension node adds a depth equal to the number of bits in its path. When
	// we reach a depth of zero again, it means we have passed the maximum depth
	// and we reached the point of insertion for leaf nodes.
	depth := uint8(0)

	// The `PathLoop` is responsible for creating all of the intermediary branch
	// and extension nodes up to the insertion point of the leaf. We start at
	// the root and insert these nodes until we reach the maximum depth. Once
	// maximum depth is reached, we know we have reached the point of insertion
	// of the leaf node.
	for {

		// We always want to keep track of the parent, so when we break out of
		// the loop, we can look at it to determine the idiosyncratic Flow leaf
		// height.
		parent = current

		// In this switch statement, we create the next intermediary node, based
		// on what we encounter on the path. After the switch statement, we
		// check whether we have reached maximum depth in order to break out of
		// the loop.
		switch node := (*current).(type) {

		// If we reach a `nil` node as part of the path traversal, it means that
		// there are no intermediary nodes left on the given path; we are
		// entering new territory. At this point, we can simply insert an
		// extension node with the remainder of the path and skip to maximum
		// depth.
		case nil:

			// We insert an extension node at the location of the current pointer,
			// which is currently empty. We already put an empty leaf as its
			// child.
			extension := Extension{
				path:  path[:],
				count: maxDepth - depth,
			}
			*current = &extension
			current = &(extension.child)

			// NOTE: `node-count` is zero-based, so a value of one still means
			// that there is one bit in the extension node's path. We thus have
			// to add `node.count+1` to accurately increase depth. This can
			// overflow, but this is fine. When we reach a depth of zero, we
			// will know that we reached the depth for leaves.
			depth += (extension.count + 1)
			break

		// If we run into a branch node, we simply forward the pointers to the
		// correct side and increase the depth.
		case *Branch:

			// If the current bit is zero, we go left; if it is one, we go right.
			if bitutils.Bit(path[:], int(depth)) == 0 {
				current = &node.left
			} else {
				current = &node.right
			}
			node.clean = false

			// NOTE: if we are at maximum depth, this will overflow and set depth
			// back to zero, which is the condition we check for to realize we
			// have to have a leaf node.
			depth++
			break

		// When we run into an extension, things become more complicated.
		// Depending on how much of the path we share with the extension node,
		// we need to do different things.
		case *Extension:

			// In all cases, we will have some kind of modification of the
			// extension or the trie part below the extension, so we mark it as
			// dirty only once.
			node.clean = false

			// The first edge case happens when we have no bits in common. We
			// handle this explicitly here, for two reasons:
			// 1) It allows us to use a zero-based `common` count of bits later,
			// where `0` corresponds to `1`, and so on, just like the `node.count`
			// of the extension node.
			// 2) We can use the existing extension for the part below the new
			// branch node, while the rest of the code uses it for the part above
			// the new branch node, thus avoiding garbage collection and allocations.
			insertionBit := bitutils.Bit(path[:], int(depth))
			extensionBit := bitutils.Bit(node.path[:], int(depth))
			if insertionBit != extensionBit {

				// First, we insert the branch and set its children correctly.
				branch := &Branch{}
				*current = branch
				if insertionBit == 0 {
					current = &(branch.left)
					branch.right = node
				} else {
					current = &(branch.right)
					branch.left = node
				}

				// TODO: check if the extension's child is a leaf, in which
				// case we need to recompute its hash.

				// Finally, we move to the next depth, which is the correct
				// child of the branch we just introduced.
				depth++
				break
			}

			// At this point, we know that we have at least one bit in common
			// with the extension's path, so a common value of zero is implicitly
			// a one. We count common bits starting the the second bit, so the
			// `common` value is zero-based, just like the `node.count` value.
			common := uint8(0)
			for i := depth + 1; i <= depth+node.count; i++ {
				if bitutils.Bit(path[:], int(i)) != bitutils.Bit(node.path[:], int(i)) {
					break
				}
				common++
			}

			// We increase the depth to point to the first node after the path
			// we have in common with the extension node. We have to add one extra
			// bit because `common` is zero-based.
			// NOTE: `depth` can overflow here, but that's behaviour we want and rely
			// on; a value of zero after the switch statement indicates that we
			// have reached the depth where leafs are located.
			depth += (common + 1)

			// If we have all of the bits in common with the extension node, we
			// can simply skip to the end of the extension node here; no
			// modifications are needed.
			if common == node.count {
				node.clean = false
				current = &node.child
				continue
			}

			// At this point we have:
			// - at least one bit in common, for which we can reuse the current
			//   extension; and
			// - at least one bit that is different, which means we have to
			// create a branch.

			// First, we have to determine what the child for the extension path
			// that is distinct from the insertion path will be. If we only have
			// a single bit that is different, we can point the branch node
			// to the child of the current extension node. Otherwise, we have
			// to insert an extension node in-between that holds the remainder
			// of the extension node's path.
			child := node.child
			if common > node.count-1 {
				child = &Extension{
					path:  node.path,
					count: node.count - common - 1,
					child: child,
				}
			}

			// TODO: whether with or without extension, we need to recompute the
			// child node's hash if it is a leaf.

			// Then, we can cut the path on the current extension node to
			// correspond to only the shared path and add the branch as its
			// child.
			branch := &Branch{}
			node.count = common
			node.child = branch

			// Finally, we determine whether the mismatching part of the path
			// goes to the left or the right of the branch.
			forkingBit := bitutils.Bit(node.path[:], int(depth))
			if forkingBit == 0 {
				branch.left = child
				current = &(branch.right)
			} else {
				branch.right = child
				current = &(branch.left)
			}

			// We have to increase depth here again, as we now already have a
			// branch node at the bit where we mismatch, and we go to the child
			// of that branch node.
			// NOTE: as usual, we can overflow here, which will break us out of
			// the path iterations, and skip to creating/changing the leaf.
			depth++
			break
		}

		// If after inserting the next intermediary node, we have a depth of
		// zero, it means `depth` has overflown and we reached the leaf node.
		if depth == 0 {
			break
		}
	}

	// TODO: make sure that we have the correct height here to calculate the
	// node's hash.
	height := uint16(0)
	switch p := (*parent).(type) {
	case *Extension:
		height = ledger.NodeMaxHeight - uint16(maxDepth-p.count)
	case *Branch:
		height = ledger.NodeMaxHeight - uint16(maxDepth)
	}

	// If there is no leaf node at the current path yet, we have to insert one,
	// including all of its field values.
	leaf, ok := (*current).(*Leaf)
	if !ok {
		*current = &Leaf{
			hash: ledger.ComputeCompactValue(trie.Hash(path), payload.Value, int(height)),
			path: path[:],
			key:  key,
		}
		return nil
	}

	// However, if there was already a leaf, we only need to update it if the
	// key has changed. Otherwise, the payload is still the same.
	if key == leaf.key {
		return nil
	}

	// Finally, if the key has changed, we recompute the hash, but we do not need
	// to update the path, which might save us some memory on duplicate insertions.
	leaf.hash = ledger.ComputeCompactValue(trie.Hash(path), payload.Value, int(height))
	leaf.key = key
	return nil
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
