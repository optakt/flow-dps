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

// TestExtension verifies that the hash value of a branch node with
// both children (left and right) is computed correctly. As it is in our implementation,
// extensions can never have less than two children, so no further test is necessary.
func TestExtension(t *testing.T) {
	t.Skip()
}
