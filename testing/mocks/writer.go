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
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Writer struct {
	FirstFunc        func(height uint64) error
	LastFunc         func(height uint64) error
	HeaderFunc       func(height uint64, header *flow.Header) error
	CommitFunc       func(height uint64, commit flow.StateCommitment) error
	EventsFunc       func(height uint64, events []flow.Event) error
	PayloadsFunc     func(height uint64, paths []ledger.Path, value []*ledger.Payload) error
	HeightFunc       func(blockID flow.Identifier, height uint64) error
	TransactionsFunc func(height uint64, transactions []*flow.TransactionBody) error
	CollectionsFunc  func(blockID flow.Identifier, collections []flow.LightCollection) error
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

func (w *Writer) Events(height uint64, events []flow.Event) error {
	return w.EventsFunc(height, events)
}

func (w *Writer) Payloads(height uint64, paths []ledger.Path, values []*ledger.Payload) error {
	return w.PayloadsFunc(height, paths, values)
}

func (w *Writer) Height(blockID flow.Identifier, height uint64) error {
	return w.HeightFunc(blockID, height)
}

func (w *Writer) Transactions(height uint64, transactions []*flow.TransactionBody) error {
	return w.TransactionsFunc(height, transactions)
}

func (w *Writer) Collections(blockID flow.Identifier, collections []flow.LightCollection) error {
	return w.CollectionsFunc(blockID, collections)
}
