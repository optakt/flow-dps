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

package execution_test

import (
	"testing"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/mempool/entity"

	"github.com/optakt/flow-dps/follower/execution"
	"github.com/optakt/flow-dps/testing/mocks"
)

// FIXME: Rewrite tests.

func fakeBlockData(t *testing.T) execution.BlockData {
	t.Helper()

	var collections []*entity.CompleteCollection
	for _, guar := range mocks.GenericGuarantees(1) {
		collections = append(collections, &entity.CompleteCollection{
			Guarantee:    guar,
			Transactions: mocks.GenericTransactions(2),
		})
	}

	var events []*flow.Event
	for _, event := range mocks.GenericEvents(4) {
		event := event
		events = append(events, &event)
	}

	blockData := execution.BlockData{
		Block: &flow.Block{
			Header: mocks.GenericHeader,
			Payload: &flow.Payload{
				Guarantees: mocks.GenericGuarantees(1),
				Seals:      mocks.GenericSeals(4),
			},
		},
		Collections: collections,
		TxResults:   mocks.GenericResults(4),
		Events:      events,
		TrieUpdates: []*ledger.TrieUpdate{mocks.GenericTrieUpdate},
	}

	return blockData
}
