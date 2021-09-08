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
	err := d.db.View(operation.RetrieveRootHeight(&height))
	if err != nil {
		return 0, fmt.Errorf("could not look up root height: %w", err)
	}
	return height, nil
}

func (d *Disk) Height(blockID flow.Identifier) (uint64, error) {

	var header flow.Header
	err := d.db.View(operation.RetrieveHeader(blockID, &header))
	if err != nil {
		return 0, fmt.Errorf("could not retrieve header: %w", err)
	}

	return header.Height, nil
}

func (d *Disk) Commit(height uint64) (flow.StateCommitment, error) {

	blockID, err := d.block(height)
	if errors.Is(err, storage.ErrNotFound) {
		return flow.DummyStateCommitment, dps.ErrFinished
	}
	if err != nil {
		return flow.DummyStateCommitment, fmt.Errorf("could not get block for height: %w", err)
	}

	var commit flow.StateCommitment
	err = d.db.View(operation.LookupStateCommitment(blockID, &commit))
	if err != nil {
		return flow.DummyStateCommitment, fmt.Errorf("could not look up commit: %w", err)
	}

	return commit, nil
}

func (d *Disk) Header(height uint64) (*flow.Header, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var header flow.Header
	err = d.db.View(operation.RetrieveHeader(blockID, &header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve header: %w", err)
	}
	return &header, nil
}

func (d *Disk) Collections(height uint64) ([]*flow.LightCollection, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var collIDs []flow.Identifier
	err = d.db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
	if err != nil {
		return nil, fmt.Errorf("could not lookup collections: %w", err)
	}

	collections := make([]*flow.LightCollection, 0, len(collIDs))
	for _, collID := range collIDs {
		var collection flow.LightCollection
		err := d.db.View(operation.RetrieveCollection(collID, &collection))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve collection (%x): %w", collID, err)
		}
		collections = append(collections, &collection)
	}

	return collections, nil
}

func (d *Disk) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var collIDs []flow.Identifier
	err = d.db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
	if err != nil {
		return nil, fmt.Errorf("could not lookup collections: %w", err)
	}

	guarantees := make([]*flow.CollectionGuarantee, 0, len(collIDs))
	for _, collID := range collIDs {
		var guarantee flow.CollectionGuarantee
		err := d.db.View(operation.RetrieveGuarantee(collID, &guarantee))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve guarantee (%x): %w", collID, err)
		}
		guarantees = append(guarantees, &guarantee)
	}

	return guarantees, nil
}

func (d *Disk) Transactions(height uint64) ([]*flow.TransactionBody, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var collIDs []flow.Identifier
	err = d.db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
	if err != nil {
		return nil, fmt.Errorf("could not lookup collections: %w", err)
	}

	var transactions []*flow.TransactionBody
	for _, collID := range collIDs {
		var collection flow.LightCollection
		err := d.db.View(operation.RetrieveCollection(collID, &collection))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve collection (%x): %w", collID, err)
		}
		for _, txID := range collection.Transactions {
			var transaction flow.TransactionBody
			err := d.db.View(operation.RetrieveTransaction(txID, &transaction))
			if err != nil {
				return nil, fmt.Errorf("could not retrieve transaction (%x): %w", txID, err)
			}
			transactions = append(transactions, &transaction)
		}
	}

	return transactions, nil
}

func (d *Disk) Results(height uint64) ([]*flow.TransactionResult, error) {
	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var results []flow.TransactionResult
	err = d.db.View(operation.LookupTransactionResultsByBlockID(blockID, &results))
	if err != nil {
		return nil, fmt.Errorf("could not lookup transaction results: %w", err)
	}

	// Convert to pointer slice for consistency.
	var converted []*flow.TransactionResult
	for _, result := range results {
		result := result
		converted = append(converted, &result)
	}

	return converted, nil
}

func (d *Disk) Seals(height uint64) ([]*flow.Seal, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	// LookupPayloadSeals() returns the IDs of all the seals in the specified block.
	// It should not be confused with LookupBlockSeal(), which returns the ID of the
	// *last* payload seal found in the block.
	var sealIDs []flow.Identifier
	err = d.db.View(operation.LookupPayloadSeals(blockID, &sealIDs))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve seal IDs: %w", err)
	}

	if len(sealIDs) == 0 {
		return nil, nil
	}

	seals := make([]*flow.Seal, 0, len(sealIDs))
	for _, s := range sealIDs {
		var seal flow.Seal
		err = d.db.View(operation.RetrieveSeal(s, &seal))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve seal: %w", err)
		}
		seals = append(seals, &seal)
	}

	return seals, nil
}

func (d *Disk) Events(height uint64) ([]flow.Event, error) {

	blockID, err := d.block(height)
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var events []flow.Event
	err = d.db.View(operation.LookupEventsByBlockID(blockID, &events))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	return events, nil
}

func (d *Disk) block(height uint64) (flow.Identifier, error) {

	if d.height == height {
		return d.blockID, nil
	}

	// The protocol state maps everything by block ID. However, finalized blocks
	// are unambiguously available by height, so we can look up which block ID
	// corresponds to the desired height.
	var blockID flow.Identifier
	err := d.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not look up block: %w", err)
	}

	d.height = height
	d.blockID = blockID

	return blockID, nil
}
