package forest

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/models/dps"
)

type LightForest struct {
	Nodes []*trie.LightNode
	Tries []*trie.LightTrie
}

func FlattenForest(f *Forest) (*LightForest, error) {
	tries, _ := f.GetTries()
	lightTries := make([]*trie.LightTrie, len(tries))
	lightNodes := []*trie.LightNode{nil} // First element needs to be nil.

	index := make(trie.IndexMap)
	index[nil] = 0

	count := uint64(1)
	for _, t := range tries {
		for itr := trie.NewNodeIterator(t); itr.Next(); {
			node := itr.Value()

			_, exists := index[node]
			if exists {
				continue
			}

			index[node] = count
			count++

			lightNode, err := trie.ToLightNode(node, index)
			if err != nil {
				return nil, fmt.Errorf("could not build light node: %w", err)
			}
			lightNodes = append(lightNodes, lightNode)
		}

		lightTrie, err := trie.ToLightTrie(t, index)
		if err != nil {
			return nil, fmt.Errorf("could not build light trie: %w", err)
		}
		lightTries = append(lightTries, lightTrie)
	}

	lightForest := LightForest{
		Nodes: lightNodes,
		Tries: lightTries,
	}

	return &lightForest, nil
}

func RebuildTries(store dps.Store, lightForest *LightForest) ([]*trie.Trie, error) {
	tries := make([]*trie.Trie, len(lightForest.Tries))

	// At this call, we give a 150 million + slice of light nodes and expect 150 million + normal nodes as the return value. As long as the `lightForest` is not garbage collected, memory usage remains huge.
	nodes, err := RebuildNodes(lightForest.Nodes)
	if err != nil {
		return nil, fmt.Errorf("could not rebuild nodes from light nodes: %w", err)
	}

	for i, lt := range lightForest.Tries {
		tr := trie.NewTrie(nodes[lt.RootIndex], store)
		rootHash := tr.RootHash()
		if !bytes.Equal(rootHash[:], lt.RootHash) {
			return nil, fmt.Errorf("restored trie root hash mismatch: %w", err)
		}

		// FIXME: We can restore a trie from a checkpoint, but if our store does not contain the
		//  payloads for this trie, the trie can't be relied upon to add new elements. Because of
		//  this, we NEED to include the payloads in the light nodes, so that the DB can be rebuilt
		//  when rebuilding the tries.

		trimmedTrie := trie.NewEmptyTrie(store)
		for _, leaf := range tr.Leaves() {
			payload, err := store.Retrieve(leaf.Hash())
			if err != nil {
				return nil, fmt.Errorf("restored trie missing payload: %w", err)
			}

			trimmedTrie.Insert(leaf.Path(), payload)
		}

		fmt.Println("Successfully rebuilt optimized trie", i)
	}

	return tries, nil
}

// RebuildNodes converts the given slice of light nodes into proper nodes.
// CAUTION: Since realistically, this function is given hundreds of millions of nodes, and returns
// similar numbers of their equivalent, formatted differently, this function deletes nodes from the
// original slice as it creates new ones, in order to let Go do garbage collection and effectively
// halving the RAM required to process a checkpoint.
// TODO: Ideally in here we'd want to directly restructure the nodes and replace unnecessary branches
//  with extensions, but that might be tricky.
func RebuildNodes(lightNodes []*trie.LightNode) ([]trie.Node, error) {
	nodes := make([]trie.Node, 0, len(lightNodes))
	for i, lightNode := range lightNodes {
		if lightNode == nil {
			nodes = append(nodes, nil)
			continue
		}

		if lightNode.LIndex >= uint64(i) || lightNode.RIndex >= uint64(i) {
			return nil, errors.New("sequence of light nodes does not satisfy descendents first relationship")
		}

		node, err := trie.FromLightNode(lightNode, nodes)
		if err != nil {
			return nil, fmt.Errorf("could not decode light node: %w", err)
		}

		nodes = append(nodes, node)
		if i % 500000 == 0 {
			fmt.Println("Successfully rebuilt node", i)
		}

		// Remove original light node from slice to preserve memory.
		lightNodes[i] = nil
	}

	return nodes, nil
}
