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
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/ledger/forest"
	"github.com/optakt/flow-dps/ledger/store"
	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/testing/helpers"
)

func TestLightForest(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	store, err := store.NewStore(zerolog.New(io.Discard), 4*1000*1000, dir)
	require.NoError(t, err)

	f := forest.New()

	trie1 := trie.NewEmptyTrie(store)
	trie2 := trie.NewEmptyTrie(store)

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

	rebuiltTries, err := forest.RebuildTries(store, lf)
	require.NoError(t, err)

	assert.Equal(t, trie1.RootHash(), rebuiltTries[0].RootHash())
	assert.Equal(t, trie2.RootHash(), rebuiltTries[1].RootHash())
}
