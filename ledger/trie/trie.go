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

	"github.com/gammazero/deque"
	"github.com/rs/zerolog"
	"lukechampine.com/blake3"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/encoding"
	"github.com/optakt/flow-dps/models/dps"
)

const maxDepth = ledger.NodeMaxHeight - 1

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
		return ledger.RootHash(ledger.GetDefaultHashForHeight(maxDepth + 1))
	}

	return t.root.Hash(maxDepth)
}

// Insert inserts multiple values into a copy of the trie, and returns that copy without mutating the
// original trie.
// TODO: Parallelize insertions? See https://github.com/optakt/flow-dps/issues/517
func (t *Trie) Insert(paths []ledger.Path, payloads []ledger.Payload) (*Trie, error) {
	if len(paths) != len(payloads) {
		return nil, fmt.Errorf("paths and payloads must be the same length")
	}
	if len(paths) == 0 {
		return t, nil
	}

	tree := t

	root := Node(nil)
	for i := range paths {
		err := tree.insert(&root, paths[i], &payloads[i])
		if err != nil {
			return nil, fmt.Errorf("failed to insert leaf: %w", err)
		}

		tree = NewTrie(t.log, root, t.store)
	}

	return tree, nil
}

// insert adds a new leaf to the trie. While doing so, since the trie is optimized, it might
// restructure the trie by adding new extensions, branches, or even moving other nodes
// to different heights along the way.
func (t *Trie) insert(root *Node, path ledger.Path, payload *ledger.Payload) error {

	// Insertions should never fail, so we can start by encoding the payload
	// data and storing it in our key-value store. We can also optimistically
	// check whether the data is already cached and count this part.
	data := encoding.EncodePayload(payload)
	key := blake3.Sum256(data)
	err := t.store.Save(key, data)
	if err != nil {
		return fmt.Errorf("could not save leaf data to store: %w", err)
	}

	// Let's do some magic for memory optimization. If we end up inserting this
	// new leaf, it means that we forged a new path at some point down the trie.
	// In all likelihood, there will be at least one extension node that wants
	// to hold part of the path. As a shortcut, we can just hold the path in the
	// leaf node once, and use a pointer for _all_ extension nodes that hold
	// part of it. That way, we only hold the actual path once, and each
	// additional reference only has 8 bytes instead of 32.
	// At the same time, if we don't insert a new leaf, it means the path already
	// existed, and we don't need to hold the new copy of it. In that case, we
	// simply drop the new leaf and store the new key and hash on the old leaf.
	leaf := &Leaf{
		path: [32]byte(path),
	}

	// The parent node is populated at the beginning of each iteration through
	// the trie, so that we know the parent of the leaf eventually, and can
	// infer the height of the leaf from it.
	var parent Node

	// The sibling node is populated if there is a leaf on an extension node
	// that is modified, which means that the sibling's hash has to be recomputed.
	// We also need to keep track of the sibling's parent, the uncle, so we can
	// infer the right height from a potential extension node ancestor.
	// We only need to bother with uncle's that are extension nodes; otherwise,
	// a branch node is implied and the height will be zero.
	var uncle Node
	var sibling *Leaf

	// Current always points at the current node for the iteration. It's the
	// pointer that we forward while iterating along the path of the insertion.
	// We can modify the trie while iterating by replacing its contents.
	prevPointer := &t.root
	newPointer := root

	// Depth keeps track of the depth that we are at in the trie. The root node
	// is at a depth of zero; every branch node adds a depth of one, while every
	// extension node adds a depth equal to the number of bits in its path. When
	// we reach a depth of zero again, it means we have passed the maximum depth,
	// and we reached the point of insertion for leaf nodes.
	depth := uint8(0)

	// The `PathLoop` is responsible for traversing through and creating missing
	// intermediary branch  and extension nodes up to the insertion point of the
	// leaf. We start at the root, traversing nodes and inserting them as needed
	// until we reach the leaf depth.
	for {

		// In this switch statement, we create the next intermediary node, based
		// on what we encounter on the path. After the switch statement, we
		// check whether we have reached maximum depth in order to break out of
		// the loop.
		switch node := (*prevPointer).(type) {

		// If we reach a `nil` node as part of the path traversal, it means that
		// there are no intermediary nodes left on the given path; we are
		// entering new territory. At this point, we can simply insert an
		// extension node with the remainder of the path and count to maximum
		// depth.
		case nil:

			// We insert an extension node at the location of the current pointer,
			// which is currently empty. We already put an empty leaf as its
			// child.
			extension := &Extension{
				path:  &leaf.path,
				count: maxDepth - depth,
			}
			*newPointer = extension
			newPointer = &(extension.child)
			prevPointer = &(extension.child)

			// NOTE: `node-count` is zero-based, so a value of one still means
			// that there is one bit in the extension node's path. We thus have
			// to add `node.count+1` to accurately increase depth. This can
			// overflow, but this is fine. When we reach a depth of zero, we
			// will know that we reached the depth for leaves.
			parent = extension
			depth += extension.count + 1
			break

		// If we run into a branch node, we simply forward the pointers to the
		// correct side and increase the depth.
		case *Branch:

			// If the current bit is zero, we go left; if it is one, we go right.
			if bitutils.Bit(path[:], int(depth)) == 0 {
				branch := &Branch{
					right: node.right,
				}
				*newPointer = branch
				newPointer = &branch.left
				prevPointer = &node.left
			} else {
				branch := &Branch{
					left: node.left,
				}
				*newPointer = branch
				newPointer = &branch.right
				prevPointer = &node.right
			}

			// NOTE: if we are at maximum depth, this will overflow and set depth
			// back to zero, which is the condition we check for to realize we
			// have to have a leaf node.
			parent = node
			depth++
			break

		// When we run into an extension, things become more complicated.
		// Depending on how much of the path we share with the extension node,
		// we need to do different things.
		case *Extension:

			// If the child of the extension is currently a leaf, we should
			// keep track of it as sibling, so we can later recompute its hash.
			// Below, each time the ancestor of the sibling is an extension node,
			// we also keep track of it as the uncle, so we can properly determine
			// the sibling's height later.
			sibling, _ = node.child.(*Leaf)

			// The first edge case happens when we have no bits in common. We
			// handle this explicitly here, for two reasons:
			// 1) It allows us to use a zero-based `common` count of bits later,
			// where `0` corresponds to `1`, and so on, just like the `node.count`
			// of the extension node, so comparisons are consistent.
			// 2) We can use the existing extension for the part below the new
			// branch node, while the rest of the code uses it for the part above
			// the new branch node, thus avoiding garbage collection and allocations.
			insertionBit := bitutils.Bit(path[:], int(depth))
			extensionBit := bitutils.Bit(node.path[:], int(depth))
			if insertionBit != extensionBit {

				// We first determine the child of the branch node on the path
				// we do NOT follow; it's either the current extension, or the
				// child of the extension if it was only one bit long. If we
				// keep the extension, we need to shorten it by one bit.
				child := Node(node)
				if node.count == 0 {
					child = node.child
				} else {
					extension := &Extension{
						path:  node.path,
						count: node.count - 1,
						child: node.child,
					}
					uncle = extension
					child = extension
				}

				// After that, we can create the branch and, depending on the
				// bit of the insertion path, we point either the right or the
				// left side to the path we do NOT follow, and load the other
				// `nil` side on the path we DO follow into our current pointer.
				branch := &Branch{}
				*newPointer = branch
				if insertionBit == 0 {
					newPointer = &(branch.left)
					prevPointer = &(branch.left)
					branch.right = child
				} else {
					newPointer = &(branch.right)
					prevPointer = &(branch.right)
					branch.left = child
				}

				if uncle == nil {
					uncle = branch
				}

				// Either way, we have to increase the depth by one, because we
				// only skipped one bit (accounting for the branch we just used
				// to replace the extension at the current location).
				parent = branch
				depth++
				break
			}

			// At this point, we know that we have at least one bit in common
			// with the extension's path, so a common value of zero is implicitly
			// a one. We count common bits starting with the second bit, so the
			// `common` value is zero-based, just like the `node.count` value.
			common := uint8(0)
			for i := depth + 1; i != 0 && i <= depth+node.count; i++ {
				if bitutils.Bit(path[:], int(i)) != bitutils.Bit(node.path[:], int(i)) {
					break
				}
				common++
			}

			// We increase the depth to point to the first node after the path
			// we have in common with the extension node. We have to add one extra
			// because `common` is zero-based.
			// NOTE: `depth` can overflow here, but that's behaviour we want and rely
			// on; a value of zero after the switch statement indicates that we
			// have reached the depth where leaves are located.
			depth += common + 1

			// If we have all the bits in common with the extension node, we
			// can simply count to the end of the extension node here; no
			// modifications are needed.
			if common == node.count {
				extension := &Extension{
					path:  node.path,
					count: node.count,
				}
				*newPointer = extension
				newPointer = &(extension.child)
				prevPointer = &(node.child)
				parent = node
				break
			}

			// At this point, we have to insert a branch node after the current
			// extension and then, depending on remaining bits after the common
			// path, we also need to put the remaining path we do NOT follow into
			// an extra extension on that branch.
			// We start by figuring out the latter part: do we need to create
			// an extension on the existing path, which we do NOT follow, or can
			// we simply point to the child of the current extension?
			// If the length of the extension's path is bigger than the length
			// of the common path plus one - to account for the branch's bit -
			// then we need to create an extension and attach the current
			// extension's child to it. Otherwise, we can simply go straight to
			// the current extension's child from the branch.
			child := node.child
			if node.count > common+1 {
				extension := &Extension{
					path:  node.path,
					count: node.count - common - 2,
					child: child,
				}
				child = extension
				uncle = extension
			}

			// Now we can re-create the original extension with the proper values to be inserted in the mutated trie.
			extension := &Extension{
				path:  node.path,
				count: common,
				child: child,
			}
			*newPointer = extension
			newPointer = &(extension.child)

			// Then, we can cut the path on the current extension's path's length
			// to only contain the common path, and point it to the branch node
			// as its child.
			branch := &Branch{}
			*newPointer = branch

			if uncle == nil {
				uncle = branch
			}

			// Finally, we point the branch's correct side to the path we do
			// NOT follow, and forward the current pointer to point at the branch's
			// side that has not been populated yet, and on which we will continue.
			forkingBit := bitutils.Bit(path[:], int(depth))
			if forkingBit == 0 {
				newPointer = &(branch.left)
				prevPointer = &(branch.left)
				branch.right = child
			} else {
				newPointer = &(branch.right)
				prevPointer = &(branch.right)
				branch.left = child
			}

			// Depth is currently pointing at the new branch node, so we have to
			// increase it by one extra bit to count past the inserted branch node
			// and continue on our path.
			// NOTE: as in all cases, `depth` can overflow here, which will break
			// us out of iteration down the trie, and on the depth of the leaf
			// nodes, where we can handle the actual insertion.
			parent = branch
			depth++
			break
		}

		// In all cases, we should end with an overflow of the `depth` value back
		// to zero after traversing all 256 bits of the insertion path. So once
		// depth reaches zero, we can break out of the loop and insert the leaf.
		if depth == 0 {
			break
		}
	}

	// Before dealing with the leaf, we will check whether we need to recompute
	// the hash of its sibling. If we have a non-nil sibling, it means the last
	// child of a split extension was a leaf, and we need to virtually "move it
	// down" to the height below the last branch node. This accounts for Flow's
	// concept of compact leaf nodes, which is essentially the combination of a
	// leaf with the extension node it descends from. Just like we kept track of
	// the sibling as potential leaf, we kept track of the uncle as potential
	// extension, so we can determine the leaf's height.
	if sibling != nil && uncle != nil {
		// Save the current position of the pointer so that we can get back to the new path after dealing
		// with the uncle and sibling.
		height := 0
		u, ok := uncle.(*Extension)
		if ok {
			height = int(u.count) + 1
		}

		value, err := t.store.Retrieve(sibling.key)
		if err != nil {
			return fmt.Errorf("could not retrieve sibling data from store: %w", err)
		}
		payload, err := encoding.DecodePayload(value)
		if err != nil {
			return fmt.Errorf("could not decode sibling payload: %w", err)
		}

		// Update the values for the uncle and sibling in the mutated trie.
		if ok {
			// If the uncle is an extension, simply point to its child.
			newPointer = &(u.child)
		} else {
			// If the uncle is not set, then it's actually the same as the parent,
			// and must be a branch. We need to set the newPointer to the opposite
			// child to the one that is getting inserted.
			b := parent.(*Branch)
			if bitutils.Bit(path[:], int(depth-1)) == 0 {
				newPointer = &(b.right)
			} else {
				newPointer = &(b.left)
			}
		}

		clone := &Leaf{
			path: sibling.path,
			key:  sibling.key,
			hash: ledger.ComputeCompactValue(sibling.path, payload.Value, height),
		}
		*newPointer = clone

		// Get back to the original node where the pointer was before we dealt with the sibling.
		switch p := parent.(type) {
		case *Branch:
			if bitutils.Bit(path[:], int(depth-1)) == 0 {
				newPointer = &(p.left)
			} else {
				newPointer = &(p.right)
			}
		case *Extension:
			newPointer = &(p.child)
		}
	}

	// We determine the height of the leaf node in the same manner as we determined
	// the height of the sibling: if the parent is a branch node, the height is zero,
	// because the branch's height is one. This means we don't need to worry about
	// the parent's type in this case, and can simply use a height of zero as default
	// value. If the parent is an extension, however, the height of the leaf
	// corresponds to the height of the extension node, which increases by one for
	// every bit in its path. As `p.count` is zero-based, we have to account for
	// one extra bit.
	height := 0
	p, ok := parent.(*Extension)
	if ok {
		height = int(p.count) + 1
	}

	// If the current leaf at this path is `nil`, we insert the new leaf at its
	// correct location. This makes sure that the insertion path array we put on
	// the leaf is kept, as it might be referenced by the path pointers in
	// extension nodes along the way.
	if *newPointer == nil {
		*newPointer = leaf
	}

	// If we get here without inserting the new leaf at the current location, it
	// means that we already had a leaf in its place, and hence the path already
	// existed in the trie. This also means that no extension node points at the
	// path array of the new leaf, and the memory will be freed because nothing
	// will point at the new leaf.
	// In both cases, we retrieve the leaf that is at the current location and
	// update its payload key and hash. We could check for redundant hashing
	// here, but a single hash is super cheap, so we don't really need to make
	// the code path more complex. In general, we won't insert the same payload
	// at the same path, so this is negligible, just like the fact we mark all
	// nodes as dirty even when we might insert a redundant payload.
	leaf = (*newPointer).(*Leaf)
	leaf.hash = ledger.ComputeCompactValue(leaf.path, payload.Value, height)
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

		// Hitting a `nil` node for a read should only be possible when the
		// root is `nil` and the trie is completely empty.
		case nil:
			return nil, ErrPathNotFound

		// If we hit a branch node, we have to sides to it, so we just forward
		// by one and go to the correct side.
		case *Branch:

			// A zero bit goes left, a one bit goes right.
			if bitutils.Bit(path[:], int(depth)) == 0 {
				current = &node.left
			} else {
				current = &node.right
			}

			// Increase depth by one to keep track of how far along we are.
			depth++
			break

		// If we hit an extension node, we have two cases:
		// - the extension path overlaps fully with ours, and we jump to its end; or
		// - the extension path mismatches ours, and there is no value for our path.
		case *Extension:

			// We simply mimic the earlier code here, so we can use the same
			// semantics of a zero-based `common` count. If we mismatch on the
			// first bit, the path is not in our trie.
			insertionBit := bitutils.Bit(path[:], int(depth))
			extensionBit := bitutils.Bit(node.path[:], int(depth))
			if insertionBit != extensionBit {
				return nil, ErrPathNotFound
			}

			// Otherwise, we compare, starting with the second bit, using a
			// `common` value of `0` as one bit in common. That means that if
			// `common` and `node.count` don't match exactly, there is at least
			// one bit of difference.
			common := uint8(0)
			for i := depth + 1; i != 0 && i <= depth+node.count; i++ {
				if bitutils.Bit(path[:], int(i)) != bitutils.Bit(node.path[:], int(i)) {
					break
				}
				common++
			}
			if common != node.count {
				return nil, ErrPathNotFound
			}

			// At this point, we have everything in common, and we can forward
			// to the child and increase the depth accordingly.
			current = &node.child
			depth += node.count + 1
			break
		}

		// Once we reach a depth of zero, it means the value has overflown, and
		// we reached the leaf node.
		if depth == 0 {
			break
		}
	}

	// At this point, we should always have a leaf, so we use it to retrieve the
	// data and decode the payload.
	leaf := (*current).(*Leaf)
	data, err := t.store.Retrieve(leaf.key)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve payload data: %w", err)
	}
	payload, err := encoding.DecodePayload(data)
	if err != nil {
		return nil, fmt.Errorf("could not decode payload data: %w", err)
	}

	return payload, nil
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
			path, err := ledger.ToPath((*n.path)[:])
			if err != nil {
				// An extension with a leaf child should always have a full path.
				panic(err)
			}
			paths = append(paths, path)

		case *Branch:
			queue.PushBack(n.left)
			queue.PushBack(n.right)
		}
	}

	return paths
}

// FIXME: Remove this before merging.

func (t *Trie) PrintSize() {
	var (
		branchNodes    uint64
		extensionNodes uint64
		leafNodes      uint64
		branchBytes    uint64
		extensionBytes uint64
		leafBytes      uint64
		payloadBytes   uint64
	)

	queue := deque.New()
	root := t.RootNode()
	if root != nil {
		queue.PushBack(root)
	}

	for queue.Len() > 0 {
		node := queue.PopBack().(Node)
		switch n := node.(type) {
		case *Extension:
			queue.PushBack(n.child)

			extensionBytes += uint64(unsafe.Sizeof(*n))
			extensionNodes++

		case *Branch:
			queue.PushBack(n.left)
			queue.PushBack(n.right)

			branchBytes += uint64(unsafe.Sizeof(*n))
			branchNodes++

		case *Leaf:
			leafBytes += uint64(unsafe.Sizeof(*n))
			leafNodes++

			payload, _ := t.store.Retrieve(n.key)
			payloadBytes += uint64(len(payload))
		}
	}

	fmt.Printf("Extensions: %d - %d bytes\n", extensionNodes, extensionBytes)
	fmt.Printf("Branches: %d - %d bytes\n", branchNodes, branchBytes)
	fmt.Printf("Leaves: %d - %d bytes\n", leafNodes, leafBytes)
	fmt.Printf("Payloads: %d bytes\n", payloadBytes)
}
