package node

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/common/utils"
)

// Node defines an Mtrie node
//
// DEFINITIONS:
//   * HEIGHT of a node v in a tree is the number of edges on the longest
//     downward path between v and a tree leaf.
//
// Conceptually, an MTrie is a sparse Merkle Trie, which has two node types:
//   * INTERIM node: has at least one child (i.e. lChild or rChild is not
//     nil). Interim nodes do not store a path and have no payload.
//   * LEAF node: has _no_ children.
// Per convention, we also consider nil as a leaf. Formally, nil is the generic
// representative for any empty (sub)-trie (i.e. a trie without allocated
// registers).
//
// Nodes are supposed to be treated as _immutable_ data structures.
type Node struct {
	// Implementation Comments:
	// Formally, a tree can hold up to 2^maxDepth number of registers. However,
	// the current implementation is designed to operate on a sparsely populated
	// tree, holding much less than 2^64 registers.

	lChild    *Node           // Left Child
	rChild    *Node           // Right Child
	height    int             // height where the Node is at
	path      ledger.Path     // the storage path (dummy value for interim nodes)
	hashValue hash.Hash       // hash value of node (cached)
	maxDepth  uint16          // captures the longest path from this node to compacted leafs in the subtree
	regCount  uint64          // number of registers allocated in the subtree
}

var Logger zerolog.Logger

func init() {
	Logger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel)
}

// NewNode creates a new Node.
// UNCHECKED requirement: combination of values must conform to
// a valid node type (see documentation of `Node` for details)
func NewNode(
	db *badger.DB,
	height int,
	lchild,
	rchild *Node,
	path ledger.Path,
	hashValue hash.Hash,
	maxDepth uint16,
	regCount uint64) *Node {

	n := &Node{
		lChild:    lchild,
		rChild:    rchild,
		height:    height,
		path:      path,
		hashValue: hashValue,
		maxDepth:  maxDepth,
		regCount:  regCount,
	}

	err := db.Update(func(tx *badger.Txn) error {
		key := fmt.Sprintf("%x/%d", path[:], height)
		fmt.Printf(">>> Storing %d length payload at key %v\n", 0, key)
		return tx.Set([]byte(key), []byte{})
	})
	if err != nil {
		panic(err) // FIXME
	}

	return n
}

// NewLeaf creates a compact leaf Node.
// UNCHECKED requirement: height must be non-negative
// UNCHECKED requirement: payload is non nil
func NewLeaf(db *badger.DB, path ledger.Path, payload []byte, height int) *Node {

	n := &Node{
		lChild:   nil,
		rChild:   nil,
		height:   height,
		path:     path,
		maxDepth: 0,
		regCount: uint64(1),
	}
	// NOTE: In order to reduce the memory load, node leaf have their hash computed and payload discarded at creation.
	// FIXME:
	n.computeAndStoreHash(payload)
	
	err := db.Update(func(tx *badger.Txn) error {
		key := fmt.Sprintf("%x/%d", path[:], height)
		fmt.Printf(">>> Storing %d length payload at key %v\n", len(payload), key)
		return tx.Set([]byte(key), payload)
	})
	if err != nil {
		panic(err) // FIXME
	}

	return n
}

// NewInterimNode creates a new Node with the provided value and no children.
// UNCHECKED requirement: lchild.height and rchild.height must be smaller than height
// UNCHECKED requirement: if lchild != nil then height = lchild.height + 1, and same for rchild
func NewInterimNode(db *badger.DB, height int, lchild, rchild *Node) *Node {
	var lMaxDepth, rMaxDepth uint16
	var lRegCount, rRegCount uint64
	if lchild != nil {
		lMaxDepth = lchild.maxDepth
		lRegCount = lchild.regCount
	}
	if rchild != nil {
		rMaxDepth = rchild.maxDepth
		rRegCount = rchild.regCount
	}

	n := &Node{
		lChild:   lchild,
		rChild:   rchild,
		height:   height,
		maxDepth: utils.MaxUint16(lMaxDepth, rMaxDepth) + 1,
		regCount: lRegCount + rRegCount,
	}

	n.computeAndStoreHash(nil)

	err := db.Update(func(tx *badger.Txn) error {
		key := fmt.Sprintf("%x/%d", n.path[:], height)
		fmt.Printf(">>> Storing %d length payload at key %v\n", 0, key)
		return tx.Set([]byte(key), []byte{})
	})
	if err != nil {
		panic(err) // FIXME
	}

	return n
}

// computeAndStoreHash computes the node's hash value and
// stores the result in the nodes internal `hashValue` field
func (n *Node) computeAndStoreHash(payload []byte) {
	n.hashValue = n.computeHash(payload)
}

// computeHash returns the hashValue of the node
func (n *Node) computeHash(payload []byte) hash.Hash {
	// check for leaf node
	if n.lChild == nil && n.rChild == nil {
		// if payload is non-nil, compute the hash based on the payload content
		if payload != nil {
			got := ledger.ComputeCompactValue(hash.Hash(n.path), payload, n.height)
			Logger.Info().Int("node_height", n.height).Bytes("value", payload).Hex("path", n.path[:]).Hex("hash", got[:]).Msg(">>> Computed hash with payload")
			return got
		}
		//// FIXME:
		if n.Hash() != hash.DummyHash {
			Logger.Info().Int("node_height", n.height).Hex("path", n.path[:]).Hex("hash", n.hashValue[:]).Msg(">>> Computed hash with preexisting hash")
			return n.Hash()
		}
		got := ledger.GetDefaultHashForHeight(n.height)
		Logger.Info().Int("node_height", n.height).Hex("path", n.path[:]).Hex("hash", got[:]).Msg(">>> Used default hash for height")
		// if payload is nil, return the default hash
		return got
	}

	// this is an interim node at least one of lChild or rChild is not nil.
	var h1, h2 hash.Hash
	if n.lChild != nil {
		h1 = n.lChild.Hash()
	} else {
		h1 = ledger.GetDefaultHashForHeight(n.height - 1)
	}

	if n.rChild != nil {
		h2 = n.rChild.Hash()
	} else {
		h2 = ledger.GetDefaultHashForHeight(n.height - 1)
	}
	got := hash.HashInterNode(h1, h2)
	Logger.Info().Int("node_height", n.height).Hex("path", n.path[:]).Hex("hash", got[:]).Msg(">>> Computed intermediary node hash")
	return got
}

// VerifyCachedHash verifies the hash of a node is valid
func verifyCachedHashRecursive(n *Node) bool {
	if n == nil {
		return true
	}
	if !verifyCachedHashRecursive(n.lChild) {
		return false
	}

	if !verifyCachedHashRecursive(n.rChild) {
		return false
	}

	computedHash := n.computeHash(nil)
	return n.hashValue == computedHash
}

// VerifyCachedHash verifies the hash of a node is valid
func (n *Node) VerifyCachedHash() bool {
	return verifyCachedHashRecursive(n)
}

// Hash returns the Node's hash value.
// Do NOT MODIFY returned slice!
func (n *Node) Hash() hash.Hash {
	return n.hashValue
}

// Height returns the Node's height.
// Per definition, the height of a node v in a tree is the number
// of edges on the longest downward path between v and a tree leaf.
func (n *Node) Height() int {
	return n.height
}

// MaxDepth returns the longest path from this node to compacted leaves in the subtree.
// in contrast to the Height, this value captures compactness of the subtrie.
func (n *Node) MaxDepth() uint16 {
	return n.maxDepth
}

// RegCount returns number of registers allocated in the subtrie of this node.
func (n *Node) RegCount() uint64 {
	return n.regCount
}

// Path returns a pointer to the Node's register storage path.
// If the node is not a leaf, the function returns `nil`.
func (n *Node) Path() *ledger.Path {
	if n.IsLeaf() {
		return &n.path
	}
	return nil
}

// LeftChild returns the Node's left child.
// Only INTERIM nodes have children.
// Do NOT MODIFY returned Node!
func (n *Node) LeftChild() *Node { return n.lChild }

// RightChild returns the Node's right child.
// Only INTERIM nodes have children.
// Do NOT MODIFY returned Node!
func (n *Node) RightChild() *Node { return n.rChild }

// IsLeaf returns true if and only if Node is a LEAF.
func (n *Node) IsLeaf() bool {
	// Per definition, a node is a leaf if and only it has no children
	return n == nil || (n.lChild == nil && n.rChild == nil)
}

// FmtStr provides formatted string representation of the Node and subtree
func (n *Node) FmtStr(prefix string, subpath string) string {
	right := ""
	if n.rChild != nil {
		right = fmt.Sprintf("\n%v", n.rChild.FmtStr(prefix+"\t", subpath+"1"))
	}
	left := ""
	if n.lChild != nil {
		left = fmt.Sprintf("\n%v", n.lChild.FmtStr(prefix+"\t", subpath+"0"))
	}
	payloadSize := 0
	// FIXME:
	//if n.payload != nil {
	//	payloadSize = n.payload.Size()
	//}
	hashStr := hex.EncodeToString(n.hashValue[:])
	hashStr = hashStr[:3] + "..." + hashStr[len(hashStr)-3:]
	return fmt.Sprintf("%v%v: (path:%v, payloadSize:%d hash:%v)[%s] (obj %p) %v %v ", prefix, n.height, n.path, payloadSize, hashStr, subpath, n, left, right)
}
