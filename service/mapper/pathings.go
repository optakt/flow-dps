package mapper

import (
	"bytes"
	"sort"

	"github.com/gammazero/deque"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/node"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
)

func allPaths(tree *trie.MTrie) []ledger.Path {

	var paths []ledger.Path

	queue := deque.New()
	root := tree.RootNode()
	if root != nil {
		queue.PushBack(root)
	}
	for queue.Len() > 0 {
		node := queue.PopBack().(*node.Node)
		if node.IsLeaf() {
			path := node.Path()
			paths = append(paths, *path)
			continue
		}
		if node.LeftChild() != nil {
			queue.PushBack(node.LeftChild())
		}
		if node.RightChild() != nil {
			queue.PushBack(node.RightChild())
		}
	}

	return paths
}

func pathsPayloads(update *ledger.TrieUpdate) ([]ledger.Path, []ledger.Payload) {
	paths := make([]ledger.Path, 0, len(update.Paths))
	lookup := make(map[ledger.Path]*ledger.Payload)
	for i, path := range update.Paths {
		_, ok := lookup[path]
		if !ok {
			paths = append(paths, path)
		}
		lookup[path] = update.Payloads[i]
	}
	sort.Slice(paths, func(i, j int) bool {
		return bytes.Compare(paths[i][:], paths[j][:]) < 0
	})
	payloads := make([]ledger.Payload, 0, len(paths))
	for _, path := range paths {
		payloads = append(payloads, *lookup[path])
	}
	return paths, payloads
}
