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

type Reader struct {
	FirstFunc                func() (uint64, error)
	LastFunc                 func() (uint64, error)
	HeightForBlockFunc       func(blockID flow.Identifier) (uint64, error)
	CommitFunc               func(height uint64) (flow.StateCommitment, error)
	HeaderFunc               func(height uint64) (*flow.Header, error)
	EventsFunc               func(height uint64, types ...flow.EventType) ([]flow.Event, error)
	ValuesFunc               func(height uint64, paths []ledger.Path) ([]ledger.Value, error)
	TransactionFunc          func(txID flow.Identifier) (*flow.TransactionBody, error)
	TransactionsByHeightFunc func(height uint64) ([]flow.Identifier, error)
}

func BaselineReader(t *testing.T) *Reader {
	t.Helper()

	r := Reader{
		FirstFunc: func() (uint64, error) {
			return GenericHeight, nil
		},
		LastFunc: func() (uint64, error) {
			return GenericHeight, nil
		},
		HeightForBlockFunc: func(blockID flow.Identifier) (uint64, error) {
			return GenericHeight, nil
		},
		CommitFunc: func(height uint64) (flow.StateCommitment, error) {
			return GenericCommits[0], nil
		},
		HeaderFunc: func(height uint64) (*flow.Header, error) {
			return GenericHeader, nil
		},
		EventsFunc: func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return GenericEvents, nil
		},
		ValuesFunc: func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			return GenericLedgerValues, nil
		},
		TransactionFunc: func(txID flow.Identifier) (*flow.TransactionBody, error) {
			return GenericTransactions[0], nil
		},
		TransactionsByHeightFunc: func(height uint64) ([]flow.Identifier, error) {
			return GenericIdentifiers, nil
		},
	}

	return &r
}

func (r *Reader) First() (uint64, error) {
	return r.FirstFunc()
}

func (r *Reader) Last() (uint64, error) {
	return r.LastFunc()
}

func (r *Reader) HeightForBlock(blockID flow.Identifier) (uint64, error) {
	return r.HeightForBlockFunc(blockID)
}

func (r *Reader) Commit(height uint64) (flow.StateCommitment, error) {
	return r.CommitFunc(height)
}

func (r *Reader) Header(height uint64) (*flow.Header, error) {
	return r.HeaderFunc(height)
}

func (r *Reader) Events(height uint64, types ...flow.EventType) ([]flow.Event, error) {
	return r.EventsFunc(height, types...)
}

func (r *Reader) Values(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
	return r.ValuesFunc(height, paths)
}

func (r *Reader) Transaction(txID flow.Identifier) (*flow.TransactionBody, error) {
	return r.TransactionFunc(txID)
}

func (r *Reader) TransactionsByHeight(height uint64) ([]flow.Identifier, error) {
	return r.TransactionsByHeightFunc(height)
}
