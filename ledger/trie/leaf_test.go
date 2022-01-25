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

// FIXME: Not having constructors for nodes means that these tests need to become
//  internal, which means that since we need Generic tries and nodes in the mocks package
//  those tests can't use the mocks package without causing an import cycle.
package trie_test

import (
	"testing"
)

// Test_ProperLeaf verifies that the hash value of a proper leaf (at height 0) is computed correctly.
func Test_ProperLeaf(t *testing.T) {
	t.Skip()
}

// Test_CompactLeaf verifies that the hash value of a compact leaf (at height > 0) is computed correctly.
// Here, we test with 16-bit keys. Hence, the max height of a compact leaf can be 16.
// We test the hash at the lowest-possible height (1), for the leaf to still be compact,
// at an intermediary height (9) and the max possible height (256).
func Test_CompactLeaf(t *testing.T) {
	t.Skip()
}
