package flattener

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/optakt/flow-dps/ledger/forest"
	"github.com/optakt/flow-dps/models/dps"

	"github.com/optakt/flow-dps/ledger/trie"
)

// TODO: Look into making this logic part of the trie itself to avoid exposing internals of
//  the trie as part of the package API (leftChild/rightChild).

// FlattenedForest represents a Forest as a flattened data structure.
// Specifically it consists of :
//   * a list of storable nodes, where references to nodes are replaced by index in the slice
//   * and a list of storable tries, each referencing their respective root node by index.
// 0 is a special index, meaning nil, but is included in this list for ease of use
// and removing would make it necessary to constantly add/subtract indexes
//
// As an important property, the nodes are listed in an order which satisfies
// Descendents-First-Relationship. The Descendents-First-Relationship has the
// following important property:
// When re-building the Trie from the sequence of nodes, one can build the trie on the fly,
// as for each node, the children have been previously encountered.
type FlattenedForest struct {
	Nodes []*StorableNode
	Tries []*StorableTrie
}

// node2indexMap maps a node pointer to the node index in the serialization
type node2indexMap map[trie.Node]uint64

// FlattenForest returns forest FlattenedForest, which contains all nodes and tries of the Forest.
// FIXME: Make sure that nodes do not have dirty hashes before flattening!
func FlattenForest(f *forest.Forest) (*FlattenedForest, error) {
	tries, err := f.GetTries()
	if err != nil {
		return nil, fmt.Errorf("cannot get cached tries root hashes: %w", err)
	}

	storableTries := make([]*StorableTrie, 0, len(tries))
	storableNodes := []*StorableNode{nil} // 0th element is nil

	// assign unique value to every node
	allNodes := make(node2indexMap)
	allNodes[nil] = 0 // 0th element is nil

	counter := uint64(1) // start from 1, as 0 marks nil
	for _, t := range tries {
		for itr := NewNodeIterator(t); itr.Next(); {
			n := itr.Value()
			// if node not in map
			if _, has := allNodes[n]; !has {
				allNodes[n] = counter
				counter++
				storableNode, err := toStorableNode(n, allNodes)
				if err != nil {
					return nil, fmt.Errorf("failed to construct storable node: %w", err)
				}
				storableNodes = append(storableNodes, storableNode)
			}
		}
		//fix root nodes indices
		// since we indexed all nodes, root must be present
		storableTrie, err := toStorableTrie(t, allNodes)
		if err != nil {
			return nil, fmt.Errorf("failed to construct storable trie: %w", err)
		}
		storableTries = append(storableTries, storableTrie)
	}

	return &FlattenedForest{
		Nodes: storableNodes,
		Tries: storableTries,
	}, nil
}

func toStorableNode(node trie.Node, indexForNode node2indexMap) (*StorableNode, error) {
	leftIndex, found := indexForNode[node.LeftChild()]
	if !found {
		hash := node.LeftChild().Hash()
		return nil, fmt.Errorf("internal error: missing node with hash %s", hex.EncodeToString(hash[:]))
	}
	rightIndex, found := indexForNode[node.RightChild()]
	if !found {
		hash := node.RightChild().Hash()
		return nil, fmt.Errorf("internal error: missing node with hash %s", hex.EncodeToString(hash[:]))
	}

	hash := node.Hash()
	// if node is a leaf, path is a slice of 32 bytes, otherwise path is nil
	var path []byte
	nodePath := node.Path()
	if nodePath != ledger.DummyPath {
		path = nodePath[:]
	}
	storableNode := &StorableNode{
		LIndex: leftIndex,
		RIndex: rightIndex,
		Height: node.Height(),
		Path:   path,
		//EncPayload: encoding.EncodePayload(node.Payload()),
		HashValue: hash[:],
	}
	return storableNode, nil
}

func toStorableTrie(trie *trie.Trie, indexForNode node2indexMap) (*StorableTrie, error) {
	rootIndex, found := indexForNode[trie.RootNode()]
	if !found {
		hash := trie.RootNode().Hash()
		return nil, fmt.Errorf("internal error: missing node with hash %s", hex.EncodeToString(hash[:]))
	}
	hash := trie.RootHash()
	storableTrie := &StorableTrie{
		RootIndex: rootIndex,
		RootHash:  hash[:],
	}

	return storableTrie, nil
}

// RebuildTries construct a forest from a storable FlattenedForest
func RebuildTries(store dps.Store, flatForest *FlattenedForest) ([]*trie.Trie, error) {
	tries := make([]*trie.Trie, 0, len(flatForest.Tries))
	nodes, err := RebuildNodes(flatForest.Nodes)
	if err != nil {
		return nil, fmt.Errorf("reconstructing nodes from storables failed: %w", err)
	}

	//restore tries
	for _, storableTrie := range flatForest.Tries {
		trie := trie.NewTrie(nodes[storableTrie.RootIndex], store)
		rootHash := trie.RootHash()
		if !bytes.Equal(storableTrie.RootHash, rootHash[:]) {
			return nil, fmt.Errorf("restoring trie failed: roothash doesn't match")
		}
		tries = append(tries, trie)
	}
	return tries, nil
}

// FIXME: Properly recreate real nodes from storable nodes.

// RebuildNodes generates a list of Nodes from a sequence of StorableNodes.
// The sequence must obey the DESCENDANTS-FIRST-RELATIONSHIP
func RebuildNodes(storableNodes []*StorableNode) ([]trie.Node, error) {
	nodes := make([]trie.Node, 0, len(storableNodes))
	for i, snode := range storableNodes {
		if snode == nil {
			nodes = append(nodes, nil)
			continue
		}
		if (snode.LIndex >= uint64(i)) || (snode.RIndex >= uint64(i)) {
			return nil, fmt.Errorf("sequence of StorableNodes does not satisfy Descendents-First-Relationship")
		}

		if len(snode.Path) > 0 {
			path, err := ledger.ToPath(snode.Path)
			if err != nil {
				return nil, fmt.Errorf("failed to decode a path of a storableNode %w", err)
			}
			//payload, err := encoding.DecodePayload(snode.EncPayload)
			//if err != nil {
			//	return nil, fmt.Errorf("failed to decode a payload for an storableNode %w", err)
			//}
			// FIXME: Create leaves from hashes
			//nodeHash, err := hash.ToHash(snode.HashValue)
			//if err != nil {
			//	return nil, fmt.Errorf("failed to decode a hash of a storableNode %w", err)
			//}
			node := trie.NewLeaf(path, nil, snode.Height) //int(snode.Height), nodes[snode.LIndex], nodes[snode.RIndex], path, payload, nodeHash, snode.MaxDepth, snode.RegCount)
			nodes = append(nodes, node)
			continue
		}
		//nodeHash, err := hash.ToHash(snode.HashValue)
		//if err != nil {
		//	return nil, fmt.Errorf("failed to decode a hash of a storableNode %w", err)
		//}
		node := trie.NewBranch(snode.Height, nil, nil) //(int(snode.Height), nodes[snode.LIndex], nodes[snode.RIndex], ledger.DummyPath, nil, nodeHash, snode.MaxDepth, snode.RegCount)
		nodes = append(nodes, node)
	}
	return nodes, nil
}
