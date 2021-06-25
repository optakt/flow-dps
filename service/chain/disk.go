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

package chain

import (
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

type Disk struct {
	db *badger.DB
}

func FromDisk(db *badger.DB) *Disk {
	d := Disk{
		db: db,
	}

	return &d
}

func (d *Disk) Root() (uint64, error) {
	var height uint64
	err := operation.RetrieveRootHeight(&height)(d.db.NewTransaction(false))
	if err != nil {
		return 0, fmt.Errorf("could not look up root height: %w", err)
	}
	return height, nil
}

func (d *Disk) Header(height uint64) (*flow.Header, error) {
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(d.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}

	var header flow.Header
	err = operation.RetrieveHeader(blockID, &header)(d.db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve header: %w", err)
	}
	return &header, nil
}

func (d *Disk) Commit(height uint64) (flow.StateCommitment, error) {
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(d.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return flow.StateCommitment{}, dps.ErrFinished
	}

	var commit flow.StateCommitment
	err = operation.LookupStateCommitment(blockID, &commit)(d.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return flow.StateCommitment{}, dps.ErrFinished
	}
	return commit, nil
}

func (d *Disk) Events(height uint64) ([]flow.Event, error) {
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(d.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}

	var events []flow.Event
	err = operation.LookupEventsByBlockID(blockID, &events)(d.db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	return events, nil
}

func (d *Disk) Transactions(height uint64) ([]flow.Transaction, error) {
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(d.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}

	var tt []flow.TransactionResult
	err = operation.LookupTransactionResultsByBlockID(blockID, &tt)(d.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}

	var transactions []flow.Transaction
	for _, t := range tt {
		var tb flow.TransactionBody
		err := operation.RetrieveTransaction(t.TransactionID, &tb)(d.db.NewTransaction(false))
		if errors.Is(err, storage.ErrNotFound) {
			return nil, dps.ErrFinished
		}

		transactions = append(transactions, flow.Transaction{TransactionBody: tb})
	}

	return transactions, nil
}

func (d *Disk) Collections(height uint64) ([]flow.LightCollection, error) {
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(d.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}

	var tt []flow.TransactionResult
	err = operation.LookupTransactionResultsByBlockID(blockID, &tt)(d.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}

	var collections []flow.LightCollection
	for _, t := range tt {
		var id flow.Identifier
		err = operation.RetrieveCollectionID(t.ID(), &id)(d.db.NewTransaction(false))
		if errors.Is(err, storage.ErrNotFound) {
			return nil, dps.ErrFinished
		}

		var collection flow.LightCollection
		err = operation.RetrieveCollection(id, &collection)(d.db.NewTransaction(false))
		if errors.Is(err, storage.ErrNotFound) {
			return nil, dps.ErrFinished
		}

		collections = append(collections, collection)
	}

	return collections, nil
}
