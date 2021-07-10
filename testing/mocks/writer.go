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

type Writer struct {
	FirstFunc        func(height uint64) error
	LastFunc         func(height uint64) error
	HeaderFunc       func(height uint64, header *flow.Header) error
	CommitFunc       func(height uint64, commit flow.StateCommitment) error
	PayloadsFunc     func(height uint64, paths []ledger.Path, value []*ledger.Payload) error
	HeightFunc       func(blockID flow.Identifier, height uint64) error
	CollectionsFunc  func(height uint64, collections []*flow.LightCollection) error
	TransactionsFunc func(height uint64, transactions []*flow.TransactionBody) error
	ResultsFunc      func(results []*flow.TransactionResult) error
	EventsFunc       func(height uint64, events []flow.Event) error
}

func BaselineWriter(t *testing.T) *Writer {
	t.Helper()

	w := Writer{
		FirstFunc: func(height uint64) error {
			return nil
		},
		LastFunc: func(height uint64) error {
			return nil
		},
		HeaderFunc: func(height uint64, header *flow.Header) error {
			return nil
		},
		CommitFunc: func(height uint64, commit flow.StateCommitment) error {
			return nil
		},
		PayloadsFunc: func(height uint64, paths []ledger.Path, value []*ledger.Payload) error {
			return nil
		},
		HeightFunc: func(blockID flow.Identifier, height uint64) error {
			return nil
		},
		CollectionsFunc: func(height uint64, collections []*flow.LightCollection) error {
			return nil
		},
		TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error {
			return nil
		},
		ResultsFunc: func(results []*flow.TransactionResult) error {
			return nil
		},
		EventsFunc: func(height uint64, events []flow.Event) error {
			return nil
		},
	}

	return &w
}

func (w *Writer) First(height uint64) error {
	return w.FirstFunc(height)
}

func (w *Writer) Last(height uint64) error {
	return w.LastFunc(height)
}

func (w *Writer) Header(height uint64, header *flow.Header) error {
	return w.HeaderFunc(height, header)
}

func (w *Writer) Commit(height uint64, commit flow.StateCommitment) error {
	return w.CommitFunc(height, commit)
}

func (w *Writer) Payloads(height uint64, paths []ledger.Path, values []*ledger.Payload) error {
	return w.PayloadsFunc(height, paths, values)
}

func (w *Writer) Height(blockID flow.Identifier, height uint64) error {
	return w.HeightFunc(blockID, height)
}

func (w *Writer) Collections(height uint64, collections []*flow.LightCollection) error {
	return w.CollectionsFunc(height, collections)
}

func (w *Writer) Transactions(height uint64, transactions []*flow.TransactionBody) error {
	return w.TransactionsFunc(height, transactions)
}

func (w *Writer) Results(results []*flow.TransactionResult) error {
	return w.ResultsFunc(results)
}

func (w *Writer) Events(height uint64, events []flow.Event) error {
	return w.EventsFunc(height, events)
}
