package trie

import (
	"io"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type Trie struct {
	root Node
}

func (t *Trie) Insert(db *badger.DB, path ledger.Path, payload *ledger.Payload) {

	current := &t.root
	depth := 0
	for {
		switch node := (*current).(type) {
		case *Branch:
			// If the key bit at the index i is a 0, move on to the left child,
			// otherwise the right child.
			if bitutils.Bit(path[:], depth) == 0 {
				current = &node.lChild
			} else {
				current = &node.rChild
			}
			depth++

		case *Extension:
			matched := CommonBits(node.path, path)

			if matched == nodeHeight(node.skip)-1 {
				// This is a special case to avoid creating an extension node that skips nothing and to use a branch
				// instead when a new leaf's path matches with all but the last bit of the extension's skipped path.

				// We create a branch and leaf which will be the new children on this extension. The extension is
				// shortened by one.
				newBranch := NewBranch(nodeHeight(matched + 1))
				newBranch.lChild = node.lChild
				newBranch.rChild = node.rChild

				newLeaf := NewLeaf(path, payload, nodeHeight(matched+1))
				_ = db.Update(func(txn *badger.Txn) error {
					return txn.Set(path[:], payload.Value)
				})

				var remain ParentNode
				if node.height - node.skip == 1 {
					// Node only skipped one depth, so instead of moving it down under the new branch, it also needs
					// to be replaced with a branch.
					remain = NewBranch(nodeHeight(matched))
				} else {
					// Shorten the extension by one.
					node.skip = node.skip+1
					remain = node
				}

				if bitutils.Bit(path[:], matched) == 0 {
					remain.SetChildren(newLeaf, newBranch)
				} else {
					remain.SetChildren(newBranch, newLeaf)
				}

				// If the extension was long enough, it is unchanged by this statement. Otherwise, it is replaced
				// with a branch.
				*current = remain
				return
			}

			if matched == nodeHeight(node.height) {
				// The extension node is skipping over a path that is needed by the new leaf.
				// It needs to be transformed into a branch and a new extension needs to be created for the
				// remaining path that used to be skipped over.

				// Create new extension which starts lower but skips to the original height and path.
				newExt := NewExtension(nodeHeight(matched+1), node.skip, node.path)
				newExt.lChild = node.lChild
				newExt.rChild = node.rChild

				// Change the current extension's skip value to only skip up to the conflict and update its path.
				node.skip = nodeHeight(matched)
				node.path = path

				// Set the children based on whether the new extension is needed on the left or right child.
				newLeaf := NewLeaf(path, payload, nodeHeight(matched+1))
				_ = db.Update(func(txn *badger.Txn) error {
					return txn.Set(path[:], payload.Value)
				})

				// Create new branch to replace current node.
				newBranch := NewBranch(node.height)
				if bitutils.Bit(path[:], matched) == 0 {
					newBranch.lChild = newLeaf
					newBranch.rChild = newExt
				} else {
					newBranch.lChild = newExt
					newBranch.rChild = newLeaf
				}

				*current = newBranch

				return
			}

			if matched < nodeHeight(node.skip) {
				// The extension node is skipping over a path that is needed by the new leaf.
				// It needs to be shortened and a new extension node is needed at the intersection
				// of both paths.

				// Create new extension which starts lower but skips to the original height and path.
				newExt := NewExtension(nodeHeight(matched+1), node.skip, node.path)
				newExt.lChild = node.lChild
				newExt.rChild = node.rChild

				// Change the current extension's skip value to only skip up to the conflict and update its path.
				node.skip = nodeHeight(matched)
				node.path = path

				// Set the children based on whether the new extension is needed on the left or right child.
				newLeaf := NewLeaf(path, payload, nodeHeight(matched+1))
				_ = db.Update(func(txn *badger.Txn) error {
					return txn.Set(path[:], payload.Value)
				})
				if bitutils.Bit(path[:], matched) == 0 {
					node.lChild = newLeaf
					node.rChild = newExt
				} else {
					node.lChild = newExt
					node.rChild = newLeaf
				}
				return
			}

			if bitutils.Bit(path[:], nodeHeight(node.skip)) == 0 {
				current = &node.lChild
			} else {
				current = &node.rChild
			}
			depth = nodeHeight(node.skip-1)

		case *Leaf:
			if node.path == path {
				// This path conflicts with a leaf, overwrite its hash using the new payload.
				node.hash = ledger.ComputeCompactValue(hash.Hash(path), payload.Value, node.height)
				return
			}

			// This leaf is currently an extension on the path on which we need to create a new leaf. Therefore, we
			// need to replace this leaf with a branch that has two children, the new leaf and the previous one.

			matched := CommonBits(node.path, path)
			// Create an extension node that skips up to the depth at which the old and
			// new leaves diverge in path.

			var newNode ParentNode
			if depth == matched {
				// Create a branch, since an extension would not skip anything here.
				newNode = NewBranch(nodeHeight(depth))
			} else {
				// Create an extension to skip over the common bits between both node paths.
				newNode = NewExtension(nodeHeight(depth), nodeHeight(matched), path)
			}

			var oldPayload ledger.Payload
			_ = db.View(func(txn *badger.Txn) error {
				it, err := txn.Get(node.path[:])
				if err != nil {
					panic(err)
				}

				oldPayload.Value, _ = it.ValueCopy(nil)
				return nil
			})
			oldLeaf := NewLeaf(node.path, &oldPayload, nodeHeight(matched+1))
			newLeaf := NewLeaf(path, payload, nodeHeight(matched+1))
			_ = db.Update(func(txn *badger.Txn) error {
				return txn.Set(path[:], payload.Value)
			})

			// Compare first different bit between existing leaf and new leaf to know which one is which child for the
			// newly created branch.
			if bitutils.Bit(path[:], matched) == 0 {
				newNode.SetChildren(newLeaf, oldLeaf)
			} else {
				newNode.SetChildren(oldLeaf, newLeaf)
			}

			*current = newNode
			return

		case nil:
			leaf := NewLeaf(path, payload, nodeHeight(depth))
			_ = db.Update(func(txn *badger.Txn) error {
				return txn.Set(path[:], payload.Value)
			})
			*current = leaf
			return
		}
	}
}

func (t *Trie) RootHash() ledger.RootHash {
	return ledger.RootHash(t.root.Hash())
}

func (t *Trie) Dump(w io.Writer) {
	t.root.Dump(w)
}

// Converts depth into Flow Go inverted height (where 256 is root).
func nodeHeight(depth int) int {
	return ledger.NodeMaxHeight - depth
}
