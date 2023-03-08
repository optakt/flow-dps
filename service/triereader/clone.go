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

package triereader

import (
	"github.com/onflow/flow-go/ledger"
)

func clone(update *ledger.TrieUpdate) *ledger.TrieUpdate {

	var hash ledger.RootHash
	copy(hash[:], update.RootHash[:])

	paths := make([]ledger.Path, 0, len(update.Paths))
	for _, path := range update.Paths {
		var dup ledger.Path
		copy(dup[:], path[:])
		paths = append(paths, dup)
	}

	payloads := make([]*ledger.Payload, 0, len(update.Payloads))
	for _, payload := range update.Payloads {
		dup := payload.DeepCopy()
		payloads = append(payloads, dup)
	}

	dup := ledger.TrieUpdate{
		RootHash: hash,
		Paths:    paths,
		Payloads: payloads,
	}

	return &dup
}
