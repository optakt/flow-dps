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
	"math"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

type Disk struct {
	db      *badger.DB
	height  uint64
	blockID flow.Identifier
}

func FromDisk(db *badger.DB) *Disk {
	d := Disk{
		db:      db,
		height:  math.MaxUint64,
		blockID: flow.ZeroID,
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

func (d *Disk) Commit(height uint64) (flow.StateCommitment, error) {

	blockID, err := d.block(height)
	if errors.Is(err, storage.ErrNotFound) {
		return flow.StateCommitment{}, dps.ErrFinished
	}
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not get block for height: %w", err)
	}

	var commit flow.StateCommitment
	err = operation.LookupStateCommitment(blockID, &commit)(d.db.NewTransaction(false))
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not look up commit: %w", err)
	}

	return commit, nil
}

func (d *Disk) Header(height uint64) (*flow.Header, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var header flow.Header
	err = operation.RetrieveHeader(blockID, &header)(d.db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve header: %w", err)
	}
	return &header, nil
}

func (d *Disk) Events(height uint64) ([]flow.Event, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var events []flow.Event
	err = operation.LookupEventsByBlockID(blockID, &events)(d.db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	return events, nil
}

func (d *Disk) Transactions(height uint64) ([]*flow.TransactionBody, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var results []flow.TransactionResult
	err = operation.LookupTransactionResultsByBlockID(blockID, &results)(d.db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve results: %w", err)
	}

	transactions := make([]*flow.TransactionBody, 0, len(results))
	for _, result := range results {
		var transaction flow.TransactionBody
		err := operation.RetrieveTransaction(result.TransactionID, &transaction)(d.db.NewTransaction(false))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve transactions: %w", err)
		}

		transactions = append(transactions, &transaction)
	}

	return transactions, nil
}

func (d *Disk) block(height uint64) (flow.Identifier, error) {

	if d.height == height {
		return d.blockID, nil
	}

	// The protocol state maps everything by block ID. However, finalized blocks
	// are unambiguously available by height, so we can look up which block ID
	// corresponds to the desired height.
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(d.db.NewTransaction(false))
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not look up block: %w", err)
	}

	d.height = height
	d.blockID = blockID

	return blockID, nil
}
