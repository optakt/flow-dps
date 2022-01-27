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

package forest

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/models/dps"
)

// LightForest is a flattened version of a forest of tries.
// It is meant to be easily encoded/decoded into checkpoints.
type LightForest struct {
	Nodes []*trie.LightNode
	Tries []*trie.LightTrie
}

// FlattenForest flattens the given forest and returns it in the form of a light forest.
func FlattenForest(f *Forest) (*LightForest, error) {
	tries := f.Trees()
	lightTries := make([]*trie.LightTrie, 0, len(tries))
	lightNodes := []*trie.LightNode{nil} // First element needs to be nil.

	index := make(trie.IndexMap)
	index[nil] = 0

	count := uint64(1)
	for _, t := range tries {
		// Iterate on the nodes of the trie, from the bottom up.
		for itr := trie.NewNodeIterator(t); itr.Next(); {
			node := itr.Value()

			// If the node is already in the index, ignore it.
			_, exists := index[node]
			if exists {
				continue
			}

			// Store the position of the node in the index.
			index[node] = count
			count++

			// Transform the node into a light node and insert it into the slice of light nodes.
			lightNode, err := trie.ToLightNode(node, index)
			if err != nil {
				return nil, fmt.Errorf("could not build light node: %w", err)
			}
			lightNodes = append(lightNodes, lightNode)
		}

		// Once each of the trie's nodes are indexed and flattened, we can create the light trie by encoding its
		// root index.
		lightTrie, err := trie.ToLightTrie(t, index)
		if err != nil {
			return nil, fmt.Errorf("could not build light trie: %w", err)
		}
		lightTries = append(lightTries, lightTrie)
	}

	// Instantiate the light forest and return it.
	lightForest := LightForest{
		Nodes: lightNodes,
		Tries: lightTries,
	}

	return &lightForest, nil
}

// RebuildTries transforms the light tries from a light forest into proper tries, while populating the given store.
func RebuildTries(log zerolog.Logger, store dps.Store, lightForest *LightForest) ([]*trie.Trie, error) {
	tries := make([]*trie.Trie, 0, len(lightForest.Tries))

	// Convert light nodes into proper nodes.
	nodes, err := RebuildNodes(lightForest.Nodes)
	if err != nil {
		return nil, fmt.Errorf("could not rebuild nodes from light nodes: %w", err)
	}

	// Iterate on each light trie, recreate it by setting its root from the slice of proper nodes, and trim it to
	// save memory usage.
	for _, lt := range lightForest.Tries {
		// Create proper trie by setting its root using the node slice.
		tr := trie.NewTrie(log, nodes[lt.RootIndex], store)
		rootHash := tr.RootHash()
		if !bytes.Equal(rootHash[:], lt.RootHash) {
			return nil, fmt.Errorf("restored trie root hash mismatch: %w", err)
		}

		// TODO: Investigate whether it is worth trimming the tries after all.
		//       See https://github.com/optakt/flow-dps/issues/521.
		tries = append(tries, tr)
	}

	return tries, nil
}

// RebuildNodes converts the given slice of light nodes into proper nodes.
// CAUTION: Since realistically, this function is given hundreds of millions of nodes, and returns
// similar numbers of their equivalent, formatted differently, this function deletes nodes from the
// original slice as it creates new ones, in order to let Go do garbage collection and effectively
// halving the RAM required to process a checkpoint.
func RebuildNodes(lightNodes []*trie.LightNode) ([]trie.Node, error) {
	nodes := make([]trie.Node, 0, len(lightNodes))
	for i, lightNode := range lightNodes {
		if lightNode == nil {
			nodes = append(nodes, nil)
			continue
		}

		// If the node has children that have a higher index than it, it means the node slice is disordered and
		// cannot be processed.
		if lightNode.LIndex >= uint64(i) || lightNode.RIndex >= uint64(i) {
			return nil, errors.New("sequence of light nodes does not satisfy descendents first relationship")
		}

		// Convert the lightNode into a proper node and append it to the returned slice of nodes.
		node, err := trie.FromLightNode(lightNode, nodes)
		if err != nil {
			return nil, fmt.Errorf("could not decode light node: %w", err)
		}
		nodes = append(nodes, node)

		// Remove original light node from slice to preserve memory.
		lightNodes[i] = nil
	}

	return nodes, nil
}
