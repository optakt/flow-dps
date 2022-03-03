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
	"bytes"

	"github.com/onflow/flow-go/ledger"
)

type sortByPath struct {
	paths []ledger.Path
	payloads []ledger.Payload
}

func (s sortByPath) Len() int {
	return len(s.paths)
}

func (s sortByPath) Swap(i, j int) {
	s.paths[i], s.paths[j] = s.paths[j], s.paths[i]
	s.payloads[i], s.payloads[j] = s.payloads[j], s.payloads[i]
}

func (s sortByPath) Less(i, j int) bool {
	return bytes.Compare(s.paths[i][:], s.paths[j][:]) > 0
}

