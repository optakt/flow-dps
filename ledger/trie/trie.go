package trie

import (
	"io"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type trie struct {
	root Node
}

func (t *trie) Insert(db *badger.DB, path ledger.Path, payload *ledger.Payload) {

	current := &t.root
	depth := 0
	for {
		switch node := (*current).(type) {
		case *Branch:
			//fmt.Printf("Found branch at height %d. lChild is %v and rChild is %v\n", nodeHeight(depth), node.lChild != nil, node.rChild != nil)

			// If the key bit at the index i is a 0, move on to the left child,
			// otherwise the right child.
			if bitutils.Bit(path[:], depth) == 0 {
				current = &node.lChild
			} else {
				current = &node.rChild
			}
			depth++

		case *Leaf:
			if node.path == path {
				//fmt.Printf("Overwriting leaf payload at path %x depth %d with payload %v!\n", path[:], nodeHeight(depth), payload)
				// This path conflicts with a leaf, overwrite its hash using the new payload.
				node.hash = ledger.ComputeCompactValue(hash.Hash(path), payload.Value, node.height)
				return
			}

			//fmt.Printf("Splitting extension leaf for path %x at depth %d!\n", path[:], nodeHeight(depth))

			// This leaf is currently an extension on the path on which we need to create a new leaf. Therefore, we
			// need to replace this leaf with a branch that has two children, the new leaf and the previous one.

			matched := CommonBits(node.path, path)
			// Create branches up to the depth at which the old and new leaves diverge in path.
			// FIXME: Instead of creating branches, use an extension and compute the hash by imagining the branches that are not there.
			var newBranch *Branch
			for ; depth < matched; depth++ {
				newBranch = NewBranch(nodeHeight(depth))
				*current = newBranch
				if bitutils.Bit(path[:], depth) == 0 {
					current = &newBranch.lChild
				} else {
					current = &newBranch.rChild
				}
			}

			newBranch = NewBranch(nodeHeight(depth))
			var oldPayload ledger.Payload
			_ = db.View(func(txn *badger.Txn) error {
				it, err := txn.Get(node.path[:])
				if err != nil {
					panic(err)
				}

				oldPayload.Value, _ = it.ValueCopy(nil)
				return nil
			})
			oldLeaf := NewLeaf(node.path, &oldPayload, nodeHeight(depth+1))
			newLeaf := NewLeaf(path, payload, nodeHeight(depth+1))
			_ = db.Update(func(txn *badger.Txn) error {
				return txn.Set(path[:], payload.Value)
			})

			// Compare first different bit between existing leaf and new leaf to know which one is which child for the
			// newly created branch.
			//fmt.Printf("Comparing %x and %x\n", oldLeaf.path[:], newLeaf.path[:])
			if bitutils.Bit(path[:], matched) == 0 {
				newBranch.lChild = newLeaf
				newBranch.rChild = oldLeaf
			} else {
				newBranch.lChild = oldLeaf
				newBranch.rChild = newLeaf
			}
			*current = newBranch
			return

		case nil:
			//fmt.Printf("Creating leaf for path %x and height %d with payload %v!\n", path[:], nodeHeight(depth), payload)
			_ = db.Update(func(txn *badger.Txn) error {
				return txn.Set(path[:], payload.Value)
			})
			leaf := NewLeaf(path, payload, nodeHeight(depth))
			*current = leaf
			return
		}
	}
}

func (t *trie) Dump(w io.Writer) {
	t.root.Dump(w)
}

// Converts depth into Flow Go inverted height (where 256 is root).
func nodeHeight(depth int) int {
	return ledger.NodeMaxHeight - depth
}

type LightTrie struct {
	leaves map[ledger.Path]hash.Hash
	buds   map[ledger.Path]*ledger.Payload
}

func (t *LightTrie) RootHash() hash.Hash {
	var trie trie

	// Add each leaf node to a temporary trie in order to compute all node hashes.
	//for path, hash := range t.leaves {
	//	trie.InsertHash(path, hash)
	//}

	// Create new leaves for each new value that was added since the last call
	// to RootHash.
	opts := badger.DefaultOptions("")
	opts.InMemory = true
	opts.Logger = nil

	db, _ := badger.Open(opts)
	defer db.Close()

	for path, payload := range t.buds {
		// Add node to trie.
		trie.Insert(db, path, payload)

		delete(t.buds, path)
	}

	return trie.root.Hash()
}
