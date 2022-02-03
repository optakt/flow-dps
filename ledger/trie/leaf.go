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
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/encoding"
)

// Leaf is what contains the values in the trie. This implementation uses nodes that are
// compacted and do not always reside at the bottom layer of the trie.
// Instead, they are inserted at the first heights where they do not conflict with others.
// This allows the trie to keep a relatively small amount of nodes, instead of having
// many nodes/extensions for each leaf in order to bring it all the way to the bottom
// of the trie.
type Leaf struct {
	dirty   bool
	hash    [32]byte
	payload [32]byte
}

// Hash returns the leaf hash.
func (l *Leaf) Hash(height uint8, path [32]byte, getPayload payloadRetriever) [32]byte {
	if l.dirty {
		l.computeHash(height, path, getPayload)
	}
	return l.hash
}

func (l *Leaf) computeHash(height uint8, path [32]byte, getPayload payloadRetriever) {
	data, err := getPayload(l.payload)
	if err != nil {
		panic(err) // FIXME: Handle error?
	}

	payload, err := encoding.DecodePayload(data)
	if err != nil {
		panic(err) // FIXME: Handle error?
	}

	// How to access the path and payload here?
	l.hash = ledger.ComputeCompactValue(path, payload.Value, int(height)+1)
	fmt.Printf("LEAF:\t%x\t+\t%x\t+\t%d\t=\t%x\n", path[:], payload.Value[:], int(height)+1, l.hash[:])
	l.dirty = false
}
