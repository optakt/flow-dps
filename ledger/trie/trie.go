package trie

import (
	"fmt"
	"io"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type Trie struct {
	root Node
	db   *badger.DB // TODO: Make DB store payloads in a cache that periodically commits on disk for performance.
}

func NewEmptyTrie(db *badger.DB) *Trie {
	t := Trie{
		root: nil,
		db:   db,
	}

	return &t
}

func NewTrie(root Node, db *badger.DB) *Trie {
	t := Trie{
		root: root,
		db:   db,
	}

	return &t
}

func (t *Trie) RootNode() Node {
	return t.root
}

func (t *Trie) RootHash() ledger.RootHash {
	return ledger.RootHash(t.root.Hash())
}

// TODO: Add method to add multiple paths and payloads at once and parallelize insertions that do not conflict.

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
				newLeaf := NewLeaf(path, payload, nodeHeight(matched+1))
				t.persist(path, payload)

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
					// The node skipped more than one depth, so we simply shorten its skip value by one.
					*current = NewExtension(node.height, node.skip+1, node.path, lChild, rChild)
				}

				return
			}

			if matched == nodeHeight(node.height) {
				// The new leaf needs to be inserted precisely at the layer up to which the extension currently skips.
				// It needs to be transformed into a branch and a new extension needs to be created for the
				// remaining path that used to be skipped over.

				// Create new extension which starts lower but skips to the original height and path.
				newExt := NewExtension(nodeHeight(matched+1), node.skip, node.path, node.lChild, node.rChild)

				// Set the children based on whether the new extension is needed on the left or right child.
				newLeaf := NewLeaf(path, payload, nodeHeight(matched+1))
				t.persist(path, payload)

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
				newExt := NewExtension(nodeHeight(matched+1), node.skip, node.path, node.lChild, node.rChild)

				// Set the children based on whether the new extension is needed on the left or right child.
				newLeaf := NewLeaf(path, payload, nodeHeight(matched+1))
				t.persist(path, payload)

				var lChild, rChild Node
				if bitutils.Bit(path[:], int(matched)) == 0 {
					lChild = newLeaf
					rChild = newExt
				} else {
					lChild = newExt
					rChild = newLeaf
				}

				// Change children, path and skipped height of the original extension by recreating it.
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
				return
			}

			// This leaf is currently at a height which conflicts with the new path that we want to insert.
			// Therefore, we need to replace this leaf with a branch or extension that has two children, the
			// new leaf and the previous one.

			matched := commonBits(node.path, path)

			// We need to fetch the payload here since the old leaf now resides at a new height and therefore its
			// hash needs to be recomputed.
			oldPayload := t.fetch(node.path)
			oldLeaf := NewLeaf(node.path, oldPayload, nodeHeight(matched+1))

			newLeaf := NewLeaf(path, payload, nodeHeight(matched+1))
			t.persist(path, payload)

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
			*current = NewLeaf(path, payload, nodeHeight(depth))
			t.persist(path, payload)
			return
		}
	}
}

// Dump outputs the list of nodes within this trie, from top to bottom, left to right, with one node per line.
// Note: This is for debugging only and should probably not be called on huge trie structures.
func (t *Trie) Dump(w io.Writer) {
	t.root.Dump(w)
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

// FIXME: Better error handling for unsafe reads.
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
			if matched <= nodeHeight(node.skip) {
				// The path we are looking for is skipped in this trie, therefore it does not exist.
				panic(fmt.Sprintf("unsafe read: path %x not found", path[:]))
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
				panic(fmt.Sprintf("unsafe read: path %x not found", path[:]))
			}

			return t.fetch(path)

		case nil:
			panic(fmt.Sprintf("unsafe read: path %x not found", path[:]))
		}
	}
}

// persist saves a payload at the given path.
// TODO: Make this use an intermediary cache and commit the payloads to disk periodically.
func (t *Trie) persist(path ledger.Path, payload *ledger.Payload) {
	// FIXME: Error handling.

	err := t.db.Update(func(txn *badger.Txn) error {
		return txn.Set(path[:], payload.Value)
	})
	if err != nil {
		panic(err)
	}
}

// persist fetches the payload for the given path.
// TODO: Make this use an intermediary cache and commit the payloads to disk periodically.
func (t *Trie) fetch(path ledger.Path) *ledger.Payload {
	// FIXME: Error handling.

	var payload ledger.Payload
	_ = t.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get(path[:])
		if err != nil {
			panic(err)
		}

		payload.Value, _ = it.ValueCopy(nil)
		return nil
	})

	return &payload
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