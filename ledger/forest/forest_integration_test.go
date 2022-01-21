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

//go:build integration

package forest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	refForest "github.com/onflow/flow-go/ledger/complete/mtrie"
	refTrie "github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/metrics"

	"github.com/optakt/flow-dps/ledger/forest"
	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestForest_InsertAndReadBatches(t *testing.T) {
	const (
		batchSize   = 50
		totalValues = 5000
	)

	// Create our forest and the reference forest.
	f := forest.New()

	refF, err := refForest.NewForest(999_999_999, &metrics.NoopCollector{}, nil)
	require.NoError(t, err)

	// Generate paths and payloads for testing.
	paths := mocks.GenericLedgerPaths(totalValues)
	payloads := mocks.GenericLedgerPayloads(totalValues)
	payloadValues := make([]ledger.Payload, 0, totalValues)
	for _, payload := range payloads {
		payloadValues = append(payloadValues, *payload)
	}

	// Create store dependency for our trie.
	store := helpers.InMemoryStore(t)
	defer store.Close()

	// Create our trie and the reference trie.
	tr := trie.NewEmptyTrie(mocks.NoopLogger, store)
	refTr := refTrie.NewEmptyMTrie()

	// Insert ledger values by batches and add the resulting tries to the forest.
	for i := batchSize; i < len(paths); i += batchSize {
		startIdx := i - batchSize

		newTr := trie.NewTrie(mocks.NoopLogger, tr.RootNode(), store)
		for j := range paths[startIdx:i] {
			newTr.Insert(paths[startIdx+j], payloads[startIdx+j])
		}

		newRefTr, err := refTrie.NewTrieWithUpdatedRegisters(refTr, paths[i-batchSize:i], payloadValues[i-batchSize:i])
		require.NoError(t, err)

		// Verify that the tries match.
		require.Equal(t, newRefTr.RootHash(), newTr.RootHash())

		hash := tr.RootHash()
		parentCommit, err := flow.ToStateCommitment(hash[:])
		require.NoError(t, err)
		f.Add(newTr, paths[i-batchSize:i], parentCommit)

		require.NoError(t, refF.AddTrie(newRefTr))

		tr = newTr
		refTr = newRefTr
	}

	// Verify that the two forests match.
	// NOTE: The FlowGo forests always start with an empty trie, which ours do not.
	// This is why we skip the empty trie in this loop.
	wantTries, err := refF.GetTries()
	require.NoError(t, err)

	for _, wantTrie := range wantTries {
		// Skip nil root node in Flow forest.
		if wantTrie.RootNode() == nil {
			continue
		}

		hash := wantTrie.RootHash()
		commit, err := flow.ToStateCommitment(hash[:])
		require.NoError(t, err)

		require.Truef(t, f.Has(commit), "commit %x not found", commit[:])
	}
}
