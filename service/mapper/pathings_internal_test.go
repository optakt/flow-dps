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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/ledger"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestPathsPayloads(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		// Forge test update with duplicate and unsorted paths.
		testUpdate := mocks.GenericTrieUpdate(0)
		testPaths := mocks.GenericLedgerPaths(6)
		testUpdate.Paths = []ledger.Path{
			testPaths[0],
			testPaths[0],
			testPaths[1],
			testPaths[2],
			testPaths[3],
			testPaths[4],
		}

		gotPaths, gotPayloads := pathsPayloads(testUpdate)

		// Expect payloads from deduplicated paths.
		wantPayloads := []*ledger.Payload{
			mocks.GenericLedgerPayload(3),
			mocks.GenericLedgerPayload(4),
			mocks.GenericLedgerPayload(5),
			mocks.GenericLedgerPayload(1),
			mocks.GenericLedgerPayload(2),
		}
		assert.Equal(t, wantPayloads, gotPayloads)

		// Verify that paths are sorted and deduplicated.
		sortedPaths := []ledger.Path{
			testPaths[2],
			testPaths[3],
			testPaths[4],
			testPaths[0],
			testPaths[1],
		}
		assert.Equalf(t, sortedPaths, gotPaths, "expected paths to be sorted alphabetically and deduplicated")
	})

	t.Run("nominal case with empty trie update", func(t *testing.T) {
		t.Parallel()

		emptyUpdate := &ledger.TrieUpdate{}

		gotPaths, gotPayloads := pathsPayloads(emptyUpdate)

		assert.Empty(t, gotPaths)
		assert.Empty(t, gotPayloads)
	})
}
