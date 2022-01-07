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

// Leaf is what contains the values in the trie. This implementation uses nodes that are
// compacted and do not always reside at the bottom layer of the trie.
// Instead, they are inserted at the first heights where they do not conflict with others.
// This allows the trie to keep a relatively small amount of nodes, instead of having
// many nodes/extensions for each leaf in order to bring it all the way to the bottom
// of the trie.
type Leaf struct {
	hash    [32]byte
	payload [32]byte
}

// Hash returns the leaf hash.
func (l *Leaf) Hash() [32]byte {
	return l.hash
}
