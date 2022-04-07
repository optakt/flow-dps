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
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"github.com/gammazero/deque"
	"golang.org/x/sync/semaphore"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
)

const maxDepth = ledger.NodeMaxHeight - 1

// Trie is a modified Patricia-Merkle Trie, which is the storage layer of the Flow ledger.
// It uses a payload store to retrieve and persist ledger payloads.
type Trie struct {
	root Node

	// TODO: Pre-allocate pools and put elements back in pool when no longer needed.
	//  See https://github.com/optakt/flow-dps/issues/519
	groups     *sync.Pool
	extensions *sync.Pool
	branches   *sync.Pool
	leaves     *sync.Pool
}

// NewEmptyTrie creates a new trie without a root node, with the given payload store.
func NewEmptyTrie() *Trie {
	t := Trie{
		root:       nil,
		groups:     &sync.Pool{New: func() interface{} { return new(Group) }},
		extensions: &sync.Pool{New: func() interface{} { return new(Extension) }},
		branches:   &sync.Pool{New: func() interface{} { return new(Branch) }},
		leaves:     &sync.Pool{New: func() interface{} { return new(Leaf) }},
	}

	return &t
}

// NewTrie creates a new trie using the given root node and payload store.
func NewTrie(root Node) *Trie {
	t := Trie{
		root:       root,
		groups:     &sync.Pool{New: func() interface{} { return new(Group) }},
		extensions: &sync.Pool{New: func() interface{} { return new(Extension) }},
		branches:   &sync.Pool{New: func() interface{} { return new(Branch) }},
		leaves:     &sync.Pool{New: func() interface{} { return new(Leaf) }},
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

	sema := semaphore.NewWeighted(int64(2*runtime.GOMAXPROCS(0) + 1))
	hash := ledger.RootHash(t.root.Hash(sema, maxDepth))

	nodes := deque.New()
	nodes.PushFront(t.root)
	for nodes.Len() > 0 {
		node := nodes.PopBack().(Node)

		switch n := node.(type) {
		case *Branch:
			nodes.PushFront(n.left)
			nodes.PushFront(n.right)
		case *Extension:
			nodes.PushFront(n.child)
		}
	}

	return hash
}

func (t *Trie) Mutate(paths []ledger.Path, payloads []ledger.Payload) (*Trie, error) {

	// If there are no paths to be inserted, we can return right away. This will
	// save us from dealing with some edge cases in the logic that follows.
	if len(paths) == 0 {
		return t, nil
	}

	// We should have the same amount of paths and payloads.
	if len(payloads) != len(paths) {
		return nil, fmt.Errorf("mismatch between path and payload size (paths: %d, payloads: %d)", len(paths), len(payloads))
	}

	// We sort the paths and payloads by path, with zero bits to the left and
	// one bits to the right. This allows us to split the paths into two groups
	// at every depth, with all paths on the left side having the next bit at
	// zero, and all paths on the right side having the next bit at one.
	sort.Sort(sortByPath{paths, payloads})

	// Create the new trie that will hold the mutated root.
	target := &Trie{
		root:       nil,
		groups:     t.groups,
		extensions: t.extensions,
		branches:   t.branches,
		leaves:     t.leaves,
	}

	// We create a queue of groups, where each group represents a set of paths
	// that have all bits in common up to the next depth of that group.
	sink := deque.New(len(paths), len(paths))

	path := paths[0]

	// The first group of paths holds all paths, checking depth zero as the first
	// depth, and using the root to determine what to do at that depth.
	group := t.groups.Get().(*Group)
	group.path = &path
	group.source.node = &t.root
	group.target.node = &target.root
	group.start = 0
	group.end = uint(len(paths))
	group.depth = 0

	// We then push that group into our queue and start consuming it.
	sink.PushFront(group)

	// We keep processing groups that are pushed onto the queue until there are
	// no groups left to be processed.
	for sink.Len() != 0 {

		// We take the next group from the queue.
		group := sink.PopBack().(*Group)

		// We split the paths for that queue on whether their next bit is zero
		// or one, and get one set of paths on the left and one on the right.
		var pivot uint
		for pivot = group.start; pivot < group.end; pivot++ {
			bit := bitutils.Bit(paths[pivot][:], int(group.depth))
			if bit == 1 {
				break
			}
		}

		// If we have arrived at this point with a leaf, insert it.
		if group.leaf {

			// We should only have a single path left.
			if group.end-group.start > 1 {
				return nil, fmt.Errorf("duplicate path (%x)", path[:])
			}

			leaf, ok := (*group.target.node).(*Leaf)
			if !ok {
				// Create a new leaf.
				leaf = t.leaves.Get().(*Leaf)
				leaf.clean = false
				leaf.path = group.path
				leaf.payload = payloads[group.start].DeepCopy()
				*group.target.node = leaf
			} else if bytes.Compare(leaf.payload.Value, payloads[0].Value) != 0 {
				// Update leaf payload.
				leaf.clean = false
				leaf.payload = payloads[group.start].DeepCopy()
			} else {
				// Touch leaf.
				leaf.clean = false
			}

			continue
		}

		// If the group target is currently `nil`, it means we are at a new part
		// of the target trie we need to build. This might _not_ be true if we are
		// on an extension that we haven't fully traversed, for example.
		if *group.target.node == nil && group.source.node != nil {

			// In any case, if we are on a new part of the target trie, the first
			// thing we do is to look at the source trie.
			switch n := (*group.source.node).(type) {

			// If the source trie has a leaf in this location, we also create a leaf
			// on the target trie. This allows us to re-use the same path, and potentially
			// even the same payload, if it hasn't changed.
			case *Leaf:

				// Clone source leaf.
				leaf := t.leaves.Get().(*Leaf)
				leaf.clean = false
				leaf.path = n.path
				leaf.payload = n.payload
				*group.target.node = leaf

			// If the source trie has a branch in this location, we must also create
			// a branch on the target trie, because we can't drop any values. If we
			// don't go into one of the two directions on the target trie, we should
			// point the new branch at the respective side of the branch on the
			// source trie in order to re-use that subtree on the mutated trie.
			case *Branch:

				// Clone source branch.
				branch := t.branches.Get().(*Branch)
				branch.clean = false
				if pivot == group.start {
					branch.left = n.left
				}
				if pivot == group.end {
					branch.right = n.right
				}
				*group.target.node = branch

			// If the source trie has an extension in this location, we will create
			// either an extension or a branch, depending on whether we go in the
			// same direction or not.
			case *Extension:

				// If we are going in the same direction to the left or the right
				// with all of our paths, simply clone the source extension.
				extBit := bitutils.Bit(n.path[:], int(group.depth))
				if (extBit == 0 && pivot == group.end) ||
					(extBit == 1 && pivot == group.start) {

					ext := t.extensions.Get().(*Extension)
					ext.clean = false
					ext.path = n.path
					ext.count = n.count - group.source.count
					*group.target.node = ext

					break
				}

				// Otherwise, we want to introduce a branch at the current bit.
				branch := t.branches.Get().(*Branch)
				branch.clean = false

				// If we are not following both directions, point the unused side
				// of the branch to the child. And if the child is a leaf, clone
				// it so that it rehashes correctly.
				if (extBit == 0 && pivot == group.start) ||
					(extBit == 1 && pivot == group.end) {

					child := n.child
					leaf, ok := child.(*Leaf)
					if ok {
						// Clone source child leaf.
						replace := t.leaves.Get().(*Leaf)
						replace.clean = false
						replace.path = leaf.path
						replace.payload = leaf.payload

						child = replace
					}

					// If we should keep bits of the source extension, clone the
					// part of it we want to keep and set the previous child as
					// its child.
					if n.count >= group.source.count+1 {
						ext := t.extensions.Get().(*Extension)
						ext.clean = false
						ext.path = n.path
						ext.child = child
						// We need to subtract the source count we already went through
						// from its original value, plus one because we're cutting one
						// bit of it to replace it with a branch.
						ext.count = n.count - (group.source.count + 1)

						child = ext
					}

					if extBit == 0 {
						branch.left = child
					} else {
						branch.right = child
					}
				}

				// Then we place it at the current location.
				*group.target.node = branch
			}
		}

		// If the target is still `nil` here, it means the source was `nil`.
		if *group.target.node == nil {

			if pivot != group.start && pivot != group.end {
				// We are going both directions, so we create a branch.
				branch := t.branches.Get().(*Branch)
				branch.clean = false
				*group.target.node = branch
			} else {
				// All elements of this group follow the same direction,
				// so we need to create an extension.
				ext := t.extensions.Get().(*Extension)
				ext.clean = false
				ext.count = maxDepth - group.depth
				ext.path = group.path

				*group.target.node = ext
			}
		}

		// If we are not at the beginning of the extension, we might have to split
		// if some paths go into the opposite direction.
		n, ok := (*group.target.node).(*Extension)
		if ok && group.target.count > 0 {

			extBit := bitutils.Bit(n.path[:], int(group.depth))

			// If the extension points left and the group only has left-bound elements,
			// or it points right and the group only has right-bound elements, or the
			// group contains elements that need to split both ways at this bit,
			// we need to split the target extension at this bit and insert a branch.
			if (extBit == 0 && pivot == group.start) ||
				(extBit == 1 && pivot == group.end) ||
				(pivot != group.start && pivot != group.end) {

				branch := &Branch{}

				if group.source.node != nil {
					s, ok := (*group.source.node).(*Extension)
					if ok {
						child := s.child
						leaf, ok := child.(*Leaf)
						if ok {
							// Clone source child leaf.
							replace := t.leaves.Get().(*Leaf)
							replace.clean = false
							replace.path = leaf.path
							replace.payload = leaf.payload

							child = replace
						}

						// If we should keep bits of the source extension, clone the
						// part of it we want to keep and set the previous child as
						// its child.
						if group.target.count != n.count {
							ext := t.extensions.Get().(*Extension)
							ext.clean = false
							ext.path = n.path
							// We need to subtract the source count we already went through
							// from its original value, plus one because we're cutting one
							// bit of it to replace it with a branch.
							ext.count = n.count - group.target.count - 1
							ext.child = child

							child = ext
						}
						if extBit == 0 && pivot == group.start {
							branch.left = child
						}
						if extBit == 1 && pivot == group.end {
							branch.right = child
						}
					}
				}

				// Since the extension got split here, we trim it by one bit.
				n.count = group.target.count - 1
				n.child = branch

				group.target.count = 0
				group.target.node = &n.child

			}
		}

		// After this check we can increase the depth. If depth is zero, we have
		// arrived at the leaves.
		group.depth++
		if group.depth == 0 {
			group.leaf = true
		}

		// At this point, the full target trie is built up to the current depth
		// for the paths on this group, so we can step forward on both the source
		// and the target trie.
		switch n := (*group.target.node).(type) {

		// If we have an extension, we are moving forward along the path of this
		// extension on the target trie.
		case *Extension:

			// If the group count is at the extension count, we have reached the
			// end of the extension, and we want to go to the child next. Otherwise,
			// we simply increase the count of bits already checked.
			if group.target.count == n.count {
				if group.source.node != nil {
					switch s := (*group.source.node).(type) {
					case *Extension:
						if group.source.count == s.count {
							group.source.node = &s.child
							group.source.count = 0
						} else {
							group.source.count++
						}
					default:
						group.source.node = nil
					}
				}

				group.target.count = 0
				group.target.node = &n.child
			} else {
				group.target.count++

				if group.source.node != nil {
					switch s := (*group.source.node).(type) {
					case *Extension:
						if group.source.count == s.count {
							group.source.node = &s.child
							group.source.count = 0
						} else {
							group.source.count++
						}
					default:
						group.source.node = nil
					}
				}
			}

			// Finally, we queue the forwarded group for processing.
			sink.PushFront(group)

		// If we have a branch, we might move forward on both paths, or on a
		// single one, depending on the structure of the source trie.
		case *Branch:

			// We make our decision based on what side of the branch we are iterating
			// down on.
			switch {

			// If the pivot is equal to the group end, we have a single group of
			// paths on the left, and we go to the left child of the branch node.
			case pivot == group.end:

				group.target.node = &n.left

				if group.source.node != nil {
					switch s := (*group.source.node).(type) {
					// If the original node was also a branch, we should point the
					// side we don't follow on the target trie to the same side of the
					// original trie.
					case *Branch:
						group.source.node = &s.left
						group.source.count = 0

					default:
						group.source.node = nil
					}
				}

				sink.PushFront(group)

			// If the pivot is equal to the group start, we have a single group of
			// paths on the right, and we go to the right child of the branch node.
			case pivot == group.start:

				group.target.node = &n.right

				if group.source.node != nil {
					switch s := (*group.source.node).(type) {
					// If the original node was also a branch, we should point the
					// side we don't follow on the target trie to the same side of the
					// original trie.
					case *Branch:
						group.source.node = &s.right
						group.source.count = 0

					default:
						group.source.node = nil
					}
				}

				sink.PushFront(group)

			// Otherwise, have to initialize a second group to go both directions.
			default:

				path := paths[pivot]

				split := t.groups.Get().(*Group)
				split.path = &path
				split.target.node = &n.right
				split.start = pivot
				split.end = group.end
				split.depth = group.depth
				split.leaf = group.leaf

				group.target.node = &n.left
				group.end = pivot

				if group.source.node != nil {
					switch s := (*group.source.node).(type) {
					case *Branch:
						// If the original was also a branch, we should forward the source
						// pointers on both groups accordingly.
						group.source.node = &s.left
						group.source.count = 0
						split.source.node = &s.right
						split.source.count = 0

					case *Extension:
						// If the source was an extension that we split, we need keep track of it to avoid
						// losing its child.
						if bitutils.Bit(s.path[:], int(group.depth-1)) == 0 {
							if group.source.count == s.count {
								group.source.node = &s.child
								group.source.count = 0
							} else {
								group.source.count++
							}
							split.source.node = nil
						} else {
							if group.source.count == s.count {
								split.source.node = &s.child
								split.source.count = 0
							} else {
								split.source.node = group.source.node
								split.source.count = group.source.count + 1
							}
							group.source.node = nil
						}

					default:
						group.source.node = nil
						split.source.node = nil
					}
				}

				sink.PushFront(group)

				sink.PushFront(split)
			}
		}
	}

	return target, nil
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

		// If we hit a branch node, we have two sides to it, so we just forward
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

	return leaf.payload, nil
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
		case *Extension:
			queue.PushBack(n.child)

		case *Branch:
			queue.PushBack(n.left)
			queue.PushBack(n.right)

		case *Leaf:
			leaves = append(leaves, n)
		}
	}

	return leaves
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
			queue.PushBack(n.child)

		case *Branch:
			queue.PushBack(n.left)
			queue.PushBack(n.right)

		case *Leaf:
			path, err := ledger.ToPath((*n.path)[:])
			if err != nil {
				// An extension with a leaf child should always have a full path.
				panic(err)
			}
			paths = append(paths, path)
		}
	}

	return paths
}
