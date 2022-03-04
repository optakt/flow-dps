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
	"sort"
	"sync"
	"unsafe"

	"github.com/gammazero/deque"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/hash"
)

const maxDepth = ledger.NodeMaxHeight - 1

// Trie is a modified Patricia-Merkle Trie, which is the storage layer of the Flow ledger.
// It uses a payload store to retrieve and persist ledger payloads.
type Trie struct {
	root Node

	groups     *sync.Pool
	extensions *sync.Pool
}

// NewEmptyTrie creates a new trie without a root node, with the given payload store.
func NewEmptyTrie() *Trie {
	t := Trie{
		root:       nil,
		groups:     &sync.Pool{New: func() interface{} { return new(Group) }},
		extensions: &sync.Pool{New: func() interface{} { return new(Extension) }},
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

func (t *Trie) Mutate(paths []ledger.Path, payloads []ledger.Payload) *Trie {

	// If there are no paths to be inserted, we can return right away. This will
	// save us from dealing with some edge cases in the logic that follows.
	if len(paths) == 0 {
		return t
	}

	// We should have the same amount of paths and payloads.
	if len(payloads) != len(paths) {
		panic("payloads mismatch paths")
	}

	// We sort the paths and payloads by path, with zero bits to the left and
	// one bits to the right. This allows us to split the paths into two groups
	// at every depth, with all paths on the left side having the next bit at
	// zero, and all paths on the right side having the next bit at one.
	sort.Sort(sortByPath{paths, payloads})

	// We create a queue of groups, where each group represents a set of paths
	// that have all bits in common up to the next depth of that group.
	queue := deque.New(len(paths), len(paths))

	// The first group of paths holds all paths, checking depth zero as the first
	// depth, and using the root to determine what to do at that depth.
	group := t.groups.Get().(*Group)
	group.paths = paths
	group.payloads = payloads
	group.depth = 0
	group.count = 0
	group.node = &t.root

	// We then push that group into our queue and start consuming it.
	queue.PushFront(group)

	// We keep processing groups that are pushed onto the queue until there are
	// no groups left to be processed.
	for queue.Len() != 0 {

		// We take the next group from the queue.
		group := queue.PopBack().(*Group)

		// We split the paths for that queue on whether their next bit is zero
		// or one, and get one set of paths on the left and one on the right.
		var pivot int
		for index, path := range group.paths {
			bit := bitutils.Bit(path[:], int(group.depth))
			if bit == 1 {
				break
			}
			pivot = index + 1
		}

		// Now, we can cut the paths into the left and the right side.
		left := group.paths[:pivot]
		right := group.paths[pivot:]

		// The following switch only handles modification of the trie. It looks
		// at what paths need to be present at the current depth to deal with
		// all paths we are trying to insert. If anything needs to be modified,
		// it will do so.
		switch n := (*group.node).(type) {

		// If the node is currently `nil`, we have to create a node.
		case nil:

			// If we have paths on both sides, we create a new branch node, which
			// we will then follow on both sides later.
			if len(left) != 0 && len(right) != 0 {
				*group.node = &Branch{}
				continue
			}

			// If we have paths only on the left side, we insert a new extension
			// node following the path on the left.
			if len(left) != 0 {
				extension := t.extensions.Get().(*Extension)
				extension.path = &left[0]
				*group.node = extension
				continue
			}

			// If we have paths only on the right side, we insert a new extension
			// node following the path on the right.
			if len(right) != 0 {
				extension := t.extensions.Get().(*Extension)
				extension.path = &right[0]
				*group.node = extension
				continue
			}

		// If the node is currently an extension, we should create a branch if
		// there are paths on the side we don't have yet.
		case *Extension:

			// We look at the first bit of the extension.
			extBit := bitutils.Bit(n.path[:], int(group.depth))

			// If the extension is on the left, and there are no paths on the
			// right, do nothing.
			if extBit == 0 && len(right) == 0 {
				continue
			}

			// If the extension is on the right, and there are no paths on the
			// left, do nothing.
			if extBit == 1 && len(left) == 0 {
				continue
			}

			// At this point, we are always going to create a branch. We link up
			// from the deepest to the shallowest node when modifying the trie.
			branch := &Branch{}

			// We have four different cases:
			// - the extension has a length of one, and is replaced by the branch;
			// - we replace the first bit of the extension with a branch;
			// - we replace the last bit of the extension with a branch;
			// - we replace a middle bit of the extension with a branch.
			switch {

			// If the count of the extension is zero, it has a length of one. In
			// that case, the extension should simply be replaced by the branch,
			// and the correct side of the branch should point at what was
			// previously the extension's child. In this case, we recycle the
			// memory of the extension node as well to minimize allocations.
			case n.count == 0:
				if extBit == 0 {
					branch.left = n.child
				} else {
					branch.right = n.child
				}
				*group.node = branch
				t.extensions.Put(n)

			// If the count on the group is zero, we have not traversed any bits
			// of the extension yet. In that case, we shorten the extension by
			// one bit at the front, point the correct side of the branch at the
			// extension and replace the extension by the branch in its previous
			// location.
			case group.count == 0:
				n.count--
				if extBit == 0 {
					branch.left = n
				} else {
					branch.right = n
				}
				*group.node = branch

			// If the count of the group is equal to the count on the extension,
			// we are looking at the last bit of the extension now. In that case,
			// we shorten the extension by one at the back, point the extension
			// at the branch, point the correct side of the branch at the
			// extension's child and then point the current group at the branch.
			case n.count == group.count:
				if extBit == 0 {
					branch.left = n.child
				} else {
					branch.right = n.child
				}
				n.child = branch
				n.count--
				group.node = &n.child

			// Finally, in all other cases, we will have one extension before and
			// one extension behind the branch.
			default:
				extension := t.extensions.Get().(*Extension)
				extension.clean = false
				extension.path = n.path
				extension.count = n.count - group.count - 1
				extension.child = n.child
				if extBit == 0 {
					branch.left = extension
				} else {
					branch.right = extension
				}
				n.child = branch
				n.count = group.count - 1
			}

			// In all of the cases, we have to reset the group count on the group
			// because we are now pointing at a branch.
			group.count = 0

			// Additionally, if the branch we don't follow is pointing at a
			// leaf, we need to rehash it.

		}

		// At this point, we are done modifying the trie at the current depth.
		// If the current depth is 255, the next node should be the leaf, and
		// we should insert the payload, instead of queuing more groups.
		if group.depth == maxDepth {

			// If there is more than one path left, panic.
			if len(group.paths) > 0 {
				panic("duplicate paths")
			}

			// If the current leaf is not initialized, do so.
			leaf, ok := (*group.node).(*Leaf)
			if !ok {
				leaf = &Leaf{}
				*group.node = leaf
			}

			// Set the payload on the leaf.
			leaf.payload = group.payloads[0]

			// Calculate the leaf hash.
			height := 0
			p, ok := group.parent.(*Extension)
			if ok {
				height = int(p.count) + 1
			}
			leaf.hash = ledger.ComputeCompactValue(hash.Hash(leaf.path), leaf.payload.Value, height)

			continue
		}

		// At this point, we are done modifying the trie.
		switch n := (*group.node).(type) {

		// If we have an extension, we always have a single group of paths.
		case *Extension:

			// If the group count is at the extension count, we have reached the
			// end of the extension and we want to go to the child next. Otherwise
			// we simply increase the count of bits already checked.
			group.depth++
			if group.count == n.count {
				group.count = 0
				group.parent = *group.node
				group.node = &n.child
			} else {
				group.count++
			}
			queue.PushFront(group)

		// If we have a branch, we might want two groups to go on.
		case *Branch:

			// Keep the new depth, so we can recycle the current group and
			// keep the code simple.
			depth := group.depth + 1
			payloads := group.payloads
			node := *group.node
			t.groups.Put(group)

			// First, we collect the current group
			if len(left) != 0 {
				group := t.groups.Get().(*Group)
				group.paths = left
				group.payloads = payloads[:pivot]
				group.depth = depth
				group.count = 0
				group.parent = node
				group.node = &n.left
				queue.PushFront(group)
			}

			// If we have paths on the right.
			if len(right) != 0 {
				group := t.groups.Get().(*Group)
				group.paths = right
				group.payloads = payloads[pivot:]
				group.depth = depth
				group.count = 0
				group.parent = node
				group.node = &n.right
				queue.PushFront(group)
			}
		}
	}

	return t
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

	return &leaf.payload, nil
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
			leafBytes += uint64(unsafe.Sizeof(*n)) + uint64(len(n.payload.Value))
			leafNodes++
		}
	}

	fmt.Printf("Extensions: %d - %d bytes\n", extensionNodes, extensionBytes)
	fmt.Printf("Branches: %d - %d bytes\n", branchNodes, branchBytes)
	fmt.Printf("Leaves: %d - %d bytes\n", leafNodes, leafBytes)
}
