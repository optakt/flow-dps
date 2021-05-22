// Copyright 2021 Alvalor S.A.
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

package main

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/mtrie/node"
)

func allPayloads(node *node.Node) []*ledger.Payload {
	if node.IsLeaf() {
		return []*ledger.Payload{node.Payload()}
	}
	var payloads []*ledger.Payload
	left := node.LeftChild()
	if left != nil {
		payloads = append(payloads, allPayloads(left)...)
	}
	right := node.RightChild()
	if right != nil {
		payloads = append(payloads, allPayloads(right)...)
	}
	return payloads
}
