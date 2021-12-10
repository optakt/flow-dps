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
	"io"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type Leaf struct {
	path   ledger.Path
	hash   hash.Hash
	height uint16
}

func NewLeaf(height uint16, path ledger.Path, payload *ledger.Payload) *Leaf {
	//fmt.Printf("GOT: Hash for leaf computed using path %x, value %v, height %d\n", path[:], payload.Value, height)
	n := Leaf{
		path:   path,
		hash:   ledger.ComputeCompactValue(hash.Hash(path), payload.Value, int(height)),
		height: height,
	}

	return &n
}

func NewLeafWithHash(height uint16, path ledger.Path, hash hash.Hash) *Leaf {
	n := Leaf{
		path:   path,
		hash:   hash,
		height: height,
	}

	return &n
}

func (l Leaf) Hash() hash.Hash {
	return l.hash
}

func (l Leaf) Height() uint16 {
	return l.height
}

func (l Leaf) Path() ledger.Path {
	return l.path
}

func (l Leaf) LeftChild() Node {
	return nil
}

func (l Leaf) RightChild() Node {
	return nil
}

func (l Leaf) Dump(w io.Writer) {
	_, err := w.Write([]byte(fmt.Sprintf("%d:\tLEAF\t%x\t%x\n", l.height, l.hash, l.path[:])))
	if err != nil {
		panic(err)
	}
}
