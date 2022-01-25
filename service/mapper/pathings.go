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

package mapper

import (
	"bytes"
	"sort"

	"github.com/onflow/flow-go/ledger"
)

func pathsPayloads(update *ledger.TrieUpdate) ([]ledger.Path, []*ledger.Payload) {
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
	payloads := make([]*ledger.Payload, 0, len(paths))
	for _, path := range paths {
		payloads = append(payloads, lookup[path])
	}
	return paths, payloads
}
