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
	"github.com/onflow/flow-go/ledger"
)

type Group struct {
	paths    []ledger.Path
	payloads []ledger.Payload
	path     *ledger.Path
	depth    uint8
	count    uint8
	node     *Node
	leaf     bool
}

func (g *Group) Reset() *Group {
	g.paths = nil
	g.payloads = nil
	g.path = nil
	g.depth = 0
	g.count = 0
	g.node = nil
	g.leaf = false
	return g
}
