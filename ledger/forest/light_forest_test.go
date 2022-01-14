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

package forest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/ledger/forest"
	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestLightForest(t *testing.T) {

	store := helpers.InMemoryStore(t)
	defer store.Close()

	f := forest.New()

	trie1 := trie.NewEmptyTrie(mocks.NoopLogger, store)
	trie2 := trie.NewEmptyTrie(mocks.NoopLogger, store)

	paths, payloads := helpers.SampleRandomRegisterWrites(helpers.NewGenerator(), 99)
	for i := range paths {
		trie1.Insert(paths[i], &payloads[i])
	}

	paths, payloads = helpers.SampleRandomRegisterWrites(helpers.NewGenerator(), 117)
	for i := range paths {
		trie1.Insert(paths[i], &payloads[i])
	}

	f.Add(trie1, nil, flow.DummyStateCommitment)
	f.Add(trie2, nil, flow.DummyStateCommitment)

	lf, err := forest.FlattenForest(f)
	require.NoError(t, err)

	rebuiltTries, err := forest.RebuildTries(mocks.NoopLogger, store, lf)
	require.NoError(t, err)

	for _, rebuiltTrie := range rebuiltTries {
		got := rebuiltTrie.RootHash()
		assert.True(t, got == trie1.RootHash() || got == trie2.RootHash())
	}
}
