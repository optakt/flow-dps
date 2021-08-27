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

package mocks

import (
	"testing"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type ExecutionFollower struct {
	UpdateFunc       func() (*ledger.TrieUpdate, error)
	HeaderFunc       func(height uint64) (*flow.Header, error)
	CollectionsFunc  func(height uint64) ([]*flow.LightCollection, error)
	GuaranteesFunc   func(height uint64) ([]*flow.CollectionGuarantee, error)
	SealsFunc        func(height uint64) ([]*flow.Seal, error)
	TransactionsFunc func(height uint64) ([]*flow.TransactionBody, error)
	ResultsFunc      func(height uint64) ([]*flow.TransactionResult, error)
	EventsFunc       func(height uint64) ([]flow.Event, error)
}

func BaselineExecutionFollower(t *testing.T) *ExecutionFollower {
	t.Helper()

	f := ExecutionFollower{
		UpdateFunc: func() (*ledger.TrieUpdate, error) {
			return GenericTrieUpdate, nil
		},
		HeaderFunc: func(height uint64) (*flow.Header, error) {
			return GenericHeader, nil
		},
		CollectionsFunc: func(height uint64) ([]*flow.LightCollection, error) {
			return GenericCollections(4), nil
		},
		GuaranteesFunc: func(height uint64) ([]*flow.CollectionGuarantee, error) {
			return GenericGuarantees(4), nil
		},
		SealsFunc: func(height uint64) ([]*flow.Seal, error) {
			return GenericSeals(4), nil
		},
		TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) {
			return GenericTransactions(4), nil
		},
		ResultsFunc: func(height uint64) ([]*flow.TransactionResult, error) {
			return GenericResults(4), nil
		},
		EventsFunc: func(height uint64) ([]flow.Event, error) {
			return GenericEvents(4), nil
		},
	}

	return &f
}

func (e *ExecutionFollower) Update() (*ledger.TrieUpdate, error) {
	return e.UpdateFunc()
}

func (e *ExecutionFollower) Header(height uint64) (*flow.Header, error) {
	return e.HeaderFunc(height)
}

func (e *ExecutionFollower) Collections(height uint64) ([]*flow.LightCollection, error) {
	return e.CollectionsFunc(height)
}

func (e *ExecutionFollower) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {
	return e.GuaranteesFunc(height)
}

func (e *ExecutionFollower) Transactions(height uint64) ([]*flow.TransactionBody, error) {
	return e.TransactionsFunc(height)
}

func (e *ExecutionFollower) Results(height uint64) ([]*flow.TransactionResult, error) {
	return e.ResultsFunc(height)
}

func (e *ExecutionFollower) Seals(height uint64) ([]*flow.Seal, error) {
	return e.SealsFunc(height)
}

func (e *ExecutionFollower) Events(height uint64) ([]flow.Event, error) {
	return e.EventsFunc(height)
}
