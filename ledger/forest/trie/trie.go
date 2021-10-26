package trie

import (
	"fmt"
	"os"
	"sync"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/bitutils"
	"github.com/optakt/flow-dps/ledger/forest/node"
)

var Logger zerolog.Logger

func init() {
	Logger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel)
}

// MTrie represents a perfect in-memory full binary Merkle tree with uniform height.
// For a detailed description of the storage model, please consult `mtrie/README.md`
//
// A MTrie is a thin wrapper around a trie's root Node. An MTrie implements the
// logic for forming MTrie-graphs from the elementary nodes. Specifically:
//   * how Nodes (graph vertices) form a Trie,
//   * how register values are read from the trie,
//   * how Merkle proofs are generated from a trie, and
//   * how a new Trie with updated values is generated.
//
// `MTrie`s are _immutable_ data structures. Updating register values is implemented through
// copy-on-write, which creates a new `MTrie`. For minimal memory consumption, all sub-tries
// that where not affected by the write operation are shared between the original MTrie
// (before the register updates) and the updated MTrie (after the register writes).
//
// DEFINITIONS and CONVENTIONS:
//   * HEIGHT of a node v in a tree is the number of edges on the longest downward path
//     between v and a tree leaf. The height of a tree is the height of its root.
//     The height of a Trie is always the height of the fully-expanded tree.
type MTrie struct {
	root *node.Node
}

// NewEmptyMTrie returns an empty Mtrie (root is nil).
func NewEmptyMTrie() *MTrie {
	return &MTrie{root: nil}
}

// IsEmpty checks if a trie is empty.
//
// An empty try doesn't mean a trie with no allocated registers.
func (mt *MTrie) IsEmpty() bool {
	return mt.root == nil
}

// NewMTrie returns a Mtrie given the root.
func NewMTrie(root *node.Node) (*MTrie, error) {
	if root != nil && root.Height() != ledger.NodeMaxHeight {
		return nil, fmt.Errorf("height of root node must be %d but is %d", ledger.NodeMaxHeight, root.Height())
	}
	return &MTrie{
		root: root,
	}, nil
}

// RootHash returns the trie's root hash.
// Concurrency safe (as Tries are immutable structures by convention)
func (mt *MTrie) RootHash() ledger.RootHash {
	if mt.IsEmpty() {
		// case of an empty trie
		return EmptyTrieRootHash()
	}
	return ledger.RootHash(mt.root.Hash())
}

// AllocatedRegCount returns the number of allocated registers in the trie.
// Concurrency safe (as Tries are immutable structures by convention)
func (mt *MTrie) AllocatedRegCount() uint64 {
	// check if trie is empty
	if mt.IsEmpty() {
		return 0
	}
	return mt.root.RegCount()
}

// MaxDepth returns the length of the longest branch from root to leaf.
// Concurrency safe (as Tries are immutable structures by convention)
func (mt *MTrie) MaxDepth() uint16 {
	if mt.IsEmpty() {
		return 0
	}
	return mt.root.MaxDepth()
}

// RootNode returns the trie's root Node.
// Concurrency safe (as Tries are immutable structures by convention)
func (mt *MTrie) RootNode() *node.Node {
	return mt.root
}

// String returns the trie's string representation.
// Concurrency safe (as Tries are immutable structures by convention)
func (mt *MTrie) String() string {
	if mt.IsEmpty() {
		return fmt.Sprintf("Empty Trie with default root hash: %x\n", mt.RootHash())
	}
	trieStr := fmt.Sprintf("Trie root hash: %x\n", mt.RootHash())
	return trieStr + mt.root.FmtStr("", "")
}

// NewTrieWithUpdatedRegisters constructs a new trie containing all registers from the parent trie.
// The key-value pairs specify the registers whose values are supposed to hold updated values
// compared to the parent trie. Constructing the new trie is done in a COPY-ON-WRITE manner:
//   * The original trie remains unchanged.
//   * subtries that remain unchanged are from the parent trie instead of copied.
// UNSAFE: method requires the following conditions to be satisfied:
//   * keys are NOT duplicated
//   * requires _all_ paths to have a length of mt.Height bits.
// CAUTION: `updatedPaths` and `updatedPayloads` are permuted IN-PLACE for optimized processing.
func NewTrieWithUpdatedRegisters(db *badger.DB, parentTrie *MTrie, updatedPaths []ledger.Path, updatedPayloads []*ledger.Payload) (*MTrie, error) {
	parentRoot := parentTrie.root
	updatedRoot := update(db, ledger.NodeMaxHeight, parentRoot, updatedPaths, updatedPayloads, nil)
	updatedTrie, err := NewMTrie(updatedRoot)
	if err != nil {
		return nil, fmt.Errorf("constructing updated trie failed: %w", err)
	}
	return updatedTrie, nil
}

// update traverses the subtree and updates the stored registers
// CAUTION: while updating, `paths` and `payloads` are permuted IN-PLACE for optimized processing.
// UNSAFE: method requires the following conditions to be satisfied:
//   * paths all share the same common prefix [0 : mt.maxHeight-1 - nodeHeight)
//     (excluding the bit at index headHeight)
//   * paths are NOT duplicated
func update(
	db *badger.DB,
	nodeHeight int, parentNode *node.Node,
	paths []ledger.Path, payloads []*ledger.Payload, compactLeaf *node.Node,
) *node.Node {
	// No new paths to write
	if len(paths) == 0 {
		// check if a compactLeaf from a higher height is still left.
		if compactLeaf != nil {
			// create a new node for the compact leaf path and payload. The old node shouldn't
			// be recycled as it is still used by the tree copy before the update.
			// FIXME:
			path := *compactLeaf.Path()
			Logger.Info().
				Int("node_height", nodeHeight).
				Hex("path", path[:]).
				Msg(">>> Creating new leaf node from compact leaf & no new paths")

			var payload []byte
			err := db.View(func(tx *badger.Txn) error {
				key := fmt.Sprintf("%x/%d", path[:], compactLeaf.Height())
				fmt.Printf(">>> Retrieving from key %v\n", key)
				item, err := tx.Get([]byte(key))
				if err != nil {
					return err
				}

				_ = item.Value(func(val []byte) error {
					payload = val
					return nil
				})

				return nil
			})
			if err != nil {
				panic(err) // FIXME
			}
			return node.NewLeaf(db, *compactLeaf.Path(), payload, nodeHeight)
			//return node.NewLeaf(*compactLeaf.Path(), compactLeaf.Payload(), nodeHeight)
		}
		Logger.Info().
			Int("node_height", nodeHeight).
			Msg(">>> Reusing parent node because no new paths")
		return parentNode
	}

	if len(paths) == 1 && parentNode == nil && compactLeaf == nil {
		Logger.Info().
			Int("node_height", nodeHeight).
			Hex("path", paths[0][:]).
			Msg(">>> Creating new leaf node without parent and with one path")
		return node.NewLeaf(db, paths[0], payloads[0].Value, nodeHeight)
	}

	if parentNode != nil && parentNode.IsLeaf() { // if we're here then compactLeaf == nil
		// check if the parent node path is among the updated paths
		found := false
		parentPath := *parentNode.Path()
		for i, p := range paths {
			if p == parentPath {
				// the case where the recursion stops: only one path to update
				if len(paths) == 1 {
					newNode := node.NewLeaf(db, paths[i], payloads[i].Value, nodeHeight)
					if newNode.Hash() != parentNode.Hash() {
						Logger.Info().
							Int("node_height", nodeHeight).
							Hex("path", paths[i][:]).
							Msg(">>> Creating new leaf node from parent with different payload")
						return newNode
					}
					// avoid creating a new node when the same payload is written
					return parentNode
				}
				// the case where the recursion carries on: len(paths)>1
				found = true
				break
			}
		}
		if !found {
			// if the parent node carries a path not included in the input path, then the parent node
			// represents a compact leaf that needs to be carried down the recursion.
			compactLeaf = parentNode
		}
	}

	// in the remaining code: the registers to update are strictly larger than 1:
	//   - either len(paths)>1
	//   - or len(paths) == 1 and compactLeaf!= nil

	// Split paths and payloads to recurse:
	// lpaths contains all paths that have `0` at the partitionIndex
	// rpaths contains all paths that have `1` at the partitionIndex
	depth := ledger.NodeMaxHeight - nodeHeight // distance to the tree root
	partitionIndex := splitByPath(paths, payloads, depth)
	lpaths, rpaths := paths[:partitionIndex], paths[partitionIndex:]
	lpayloads, rpayloads := payloads[:partitionIndex], payloads[partitionIndex:]

	// check if there is a compact leaf that needs to get deep to height 0
	var lcompactLeaf, rcompactLeaf *node.Node
	if compactLeaf != nil {
		// if yes, check which branch it will go to.
		path := *compactLeaf.Path()
		if bitutils.Bit(path[:], depth) == 0 {
			lcompactLeaf = compactLeaf
		} else {
			rcompactLeaf = compactLeaf
		}
	}

	// set the parent node children
	var lchildParent, rchildParent *node.Node
	if parentNode != nil {
		lchildParent = parentNode.LeftChild()
		rchildParent = parentNode.RightChild()
	}

	// recurse over each branch
	var lChild, rChild *node.Node
	parallelRecursionThreshold := 16
	if len(lpaths) < parallelRecursionThreshold || len(rpaths) < parallelRecursionThreshold {
		// runtime optimization: if there are _no_ updates for either left or right sub-tree, proceed single-threaded
		lChild = update(db, nodeHeight-1, lchildParent, lpaths, lpayloads, lcompactLeaf)
		rChild = update(db, nodeHeight-1, rchildParent, rpaths, rpayloads, rcompactLeaf)
	} else {
		// runtime optimization: process the left child is a separate thread
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			lChild = update(db, nodeHeight-1, lchildParent, lpaths, lpayloads, lcompactLeaf)
		}()
		rChild = update(db, nodeHeight-1, rchildParent, rpaths, rpayloads, rcompactLeaf)
		wg.Wait()
	}

	// mitigate storage exhaustion attack: avoids creating a new node when the exact same
	// payload is re-written at a register.
	if lChild == lchildParent && rChild == rchildParent {
		return parentNode
	}
	return node.NewInterimNode(db, nodeHeight, lChild, rChild)
}

// addSiblingTrieHashToProofs inspects the sibling Trie and adds its root hash
// to the proofs, if the trie contains non-empty registers (i.e. the
// siblingTrie has a non-default hash).
func addSiblingTrieHashToProofs(siblingTrie *node.Node, depth int, proofs []*ledger.TrieProof) {
	if siblingTrie == nil || len(proofs) == 0 {
		return
	}

	// This code is necessary, because we do not remove nodes from the trie
	// when a register is deleted. Instead, we just set the respective leaf's
	// payload to empty. While this will cause the lead's hash to become the
	// default hash, the node itself remains as part of the trie.
	// However, a proof has the convention that the hash of the sibling trie
	// should only be included, if it is _non-default_. Therefore, we can
	// neither use `siblingTrie == nil` nor `siblingTrie.RegisterCount == 0`,
	// as the sibling trie might contain leaves with default value (which are
	// still counted as occupied registers)

	nodeHash := siblingTrie.Hash()
	isDef := nodeHash == ledger.GetDefaultHashForHeight(siblingTrie.Height())
	if !isDef { // in proofs, we only provide non-default value hashes
		for _, p := range proofs {
			bitutils.SetBit(p.Flags, depth)
			p.Interims = append(p.Interims, nodeHash)
		}
	}
}

// Equals compares two tries for equality.
// Tries are equal iff they store the same data (i.e. root hash matches)
// and their number and height are identical
func (mt *MTrie) Equals(o *MTrie) bool {
	if o == nil {
		return false
	}
	return o.RootHash() == mt.RootHash()
}

// EmptyTrieRootHash returns the rootHash of an empty Trie for the specified path size [bytes]
func EmptyTrieRootHash() ledger.RootHash {
	return ledger.RootHash(ledger.GetDefaultHashForHeight(ledger.NodeMaxHeight))
}

// IsAValidTrie verifies the content of the trie for potential issues
func (mt *MTrie) IsAValidTrie() bool {
	return mt.root.VerifyCachedHash()
}

// splitByPath permutes the input paths to be partitioned into 2 parts. The first part contains paths with a zero bit
// at the input bitIndex, the second part contains paths with a one at the bitIndex. The index of partition
// is returned. The same permutation is applied to the payloads slice.
//
// This would be the partition step of an ascending quick sort of paths (lexicographic order)
// with the pivot being the path with all zeros and 1 at bitIndex.
// The comparison of paths is only based on the bit at bitIndex, the function therefore assumes all paths have
// equal bits from 0 to bitIndex-1
//
//  For instance, if `paths` contains the following 3 paths, and bitIndex is `1`:
//  [[0,0,1,1], [0,1,0,1], [0,0,0,1]]
//  then `splitByPath` returns 1 and updates `paths` into:
//  [[0,0,1,1], [0,0,0,1], [0,1,0,1]]
func splitByPath(paths []ledger.Path, payloads []*ledger.Payload, bitIndex int) int {
	i := 0
	for j, path := range paths {
		bit := bitutils.Bit(path[:], bitIndex)
		if bit == 0 {
			paths[i], paths[j] = paths[j], paths[i]
			payloads[i], payloads[j] = payloads[j], payloads[i]
			i++
		}
	}
	return i
}

// SplitPaths permutes the input paths to be partitioned into 2 parts. The first part contains paths with a zero bit
// at the input bitIndex, the second part contains paths with a one at the bitIndex. The index of partition
// is returned.
//
// This would be the partition step of an ascending quick sort of paths (lexicographic order)
// with the pivot being the path with all zeros and 1 at bitIndex.
// The comparison of paths is only based on the bit at bitIndex, the function therefore assumes all paths have
// equal bits from 0 to bitIndex-1
func SplitPaths(paths []ledger.Path, bitIndex int) int {
	i := 0
	for j, path := range paths {
		bit := bitutils.Bit(path[:], bitIndex)
		if bit == 0 {
			paths[i], paths[j] = paths[j], paths[i]
			i++
		}
	}
	return i
}

// splitTrieProofsByPath permutes the input paths to be partitioned into 2 parts. The first part contains paths
// with a zero bit at the input bitIndex, the second part contains paths with a one at the bitIndex. The index
// of partition is returned. The same permutation is applied to the proofs slice.
//
// This would be the partition step of an ascending quick sort of paths (lexicographic order)
// with the pivot being the path with all zeros and 1 at bitIndex.
// The comparison of paths is only based on the bit at bitIndex, the function therefore assumes all paths have
// equal bits from 0 to bitIndex-1
func splitTrieProofsByPath(paths []ledger.Path, proofs []*ledger.TrieProof, bitIndex int) int {
	i := 0
	for j, path := range paths {
		bit := bitutils.Bit(path[:], bitIndex)
		if bit == 0 {
			paths[i], paths[j] = paths[j], paths[i]
			proofs[i], proofs[j] = proofs[j], proofs[i]
			i++
		}
	}
	return i
}
