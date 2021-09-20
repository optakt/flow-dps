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
	CollectionFunc           func(collID flow.Identifier) (*flow.LightCollection, error)
	CollectionsByHeightFunc  func(height uint64) ([]flow.Identifier, error)
	GuaranteeFunc            func(collID flow.Identifier) (*flow.CollectionGuarantee, error)
	TransactionFunc          func(txID flow.Identifier) (*flow.TransactionBody, error)
	HeightForTransactionFunc func(txID flow.Identifier) (uint64, error)
	TransactionsByHeightFunc func(height uint64) ([]flow.Identifier, error)
	ResultFunc               func(txID flow.Identifier) (*flow.TransactionResult, error)
	SealFunc                 func(sealID flow.Identifier) (*flow.Seal, error)
	SealsByHeightFunc        func(height uint64) ([]flow.Identifier, error)
	UpdatesFunc              func(height uint64) ([]ledger.Path, []*ledger.Payload, error)
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
			return GenericCommit(0), nil
		},
		HeaderFunc: func(height uint64) (*flow.Header, error) {
			return GenericHeader, nil
		},
		EventsFunc: func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return GenericEvents(4, GenericEventTypes(2)...), nil
		},
		ValuesFunc: func(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
			return GenericLedgerValues(6), nil
		},
		CollectionFunc: func(collID flow.Identifier) (*flow.LightCollection, error) {
			return GenericCollection(0), nil
		},
		CollectionsByHeightFunc: func(height uint64) ([]flow.Identifier, error) {
			return GenericCollectionIDs(5), nil
		},
		GuaranteeFunc: func(collID flow.Identifier) (*flow.CollectionGuarantee, error) {
			return GenericGuarantee(0), nil
		},
		TransactionFunc: func(txID flow.Identifier) (*flow.TransactionBody, error) {
			return GenericTransaction(0), nil
		},
		HeightForTransactionFunc: func(blockID flow.Identifier) (uint64, error) {
			return GenericHeight, nil
		},
		TransactionsByHeightFunc: func(height uint64) ([]flow.Identifier, error) {
			return GenericTransactionIDs(5), nil
		},
		ResultFunc: func(txID flow.Identifier) (*flow.TransactionResult, error) {
			return GenericResult(0), nil
		},
		SealFunc: func(sealID flow.Identifier) (*flow.Seal, error) {
			return GenericSeal(0), nil
		},
		SealsByHeightFunc: func(height uint64) ([]flow.Identifier, error) {
			return GenericSealIDs(5), nil
		},
		UpdatesFunc: func(height uint64) ([]ledger.Path, []*ledger.Payload, error) {
			return GenericLedgerPaths(5), GenericLedgerPayloads(5), nil
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

func (r *Reader) Collection(collID flow.Identifier) (*flow.LightCollection, error) {
	return r.CollectionFunc(collID)
}

func (r *Reader) CollectionsByHeight(height uint64) ([]flow.Identifier, error) {
	return r.CollectionsByHeightFunc(height)
}

func (r *Reader) Guarantee(collID flow.Identifier) (*flow.CollectionGuarantee, error) {
	return r.GuaranteeFunc(collID)
}

func (r *Reader) Transaction(txID flow.Identifier) (*flow.TransactionBody, error) {
	return r.TransactionFunc(txID)
}

func (r *Reader) HeightForTransaction(txID flow.Identifier) (uint64, error) {
	return r.HeightForTransactionFunc(txID)
}

func (r *Reader) TransactionsByHeight(height uint64) ([]flow.Identifier, error) {
	return r.TransactionsByHeightFunc(height)
}

func (r *Reader) Result(txID flow.Identifier) (*flow.TransactionResult, error) {
	return r.ResultFunc(txID)
}

func (r *Reader) Seal(sealID flow.Identifier) (*flow.Seal, error) {
	return r.SealFunc(sealID)
}

func (r *Reader) SealsByHeight(height uint64) ([]flow.Identifier, error) {
	return r.SealsByHeightFunc(height)
}

func (r *Reader) Updates(height uint64) ([]ledger.Path, []*ledger.Payload, error) {
	return r.UpdatesFunc(height)
}
