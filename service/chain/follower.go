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
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

type executionFollower interface {
	Height() uint64
}

type Follower struct {
	log zerolog.Logger

	db       *badger.DB
	follower executionFollower

	blockID flow.Identifier
	height  uint64
}

func FromFollower(log zerolog.Logger, follower executionFollower, db *badger.DB) *Follower {
	f := Follower{
		log: log,

		follower: follower,
		db:       db,

		blockID: flow.ZeroID,
	}

	return &f
}

func (f *Follower) Root() (uint64, error) {
	var height uint64
	err := f.db.View(operation.RetrieveRootHeight(&height))
	if err != nil {
		return 0, fmt.Errorf("could not look up root height: %w", err)
	}
	return height, nil
}

func (f *Follower) Commit(height uint64) (flow.StateCommitment, error) {

	blockID, err := f.block(height)
	if errors.Is(err, storage.ErrNotFound) {
		return flow.StateCommitment{}, dps.ErrFinished
	}
	if errors.Is(err, dps.ErrTimeout) {
		return flow.StateCommitment{}, dps.ErrTimeout
	}
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not get block for height: %w", err)
	}

	var commit flow.StateCommitment
	err = f.db.View(operation.LookupStateCommitment(blockID, &commit))
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not look up commit: %w", err)
	}

	return commit, nil
}

func (f *Follower) Header(height uint64) (*flow.Header, error) {

	blockID, err := f.block(height)
	if errors.Is(err, dps.ErrTimeout) {
		return nil, dps.ErrTimeout
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var header flow.Header
	err = f.db.View(operation.RetrieveHeader(blockID, &header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve header: %w", err)
	}
	return &header, nil
}

func (f *Follower) Collections(height uint64) ([]*flow.LightCollection, error) {

	blockID, err := f.block(height)
	if errors.Is(err, dps.ErrTimeout) {
		return nil, dps.ErrTimeout
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var collIDs []flow.Identifier
	err = f.db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
	if err != nil {
		return nil, fmt.Errorf("could not lookup collections: %w", err)
	}

	collections := make([]*flow.LightCollection, 0, len(collIDs))
	for _, collID := range collIDs {
		var collection flow.LightCollection
		err := f.db.View(operation.RetrieveCollection(collID, &collection))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve collection (%x): %w", collID, err)
		}
		collections = append(collections, &collection)
	}

	return collections, nil
}

func (f *Follower) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {

	blockID, err := f.block(height)
	if errors.Is(err, dps.ErrTimeout) {
		return nil, dps.ErrTimeout
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var collIDs []flow.Identifier
	err = f.db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
	if err != nil {
		return nil, fmt.Errorf("could not lookup collections: %w", err)
	}

	guarantees := make([]*flow.CollectionGuarantee, 0, len(collIDs))
	for _, collID := range collIDs {
		var guarantee flow.CollectionGuarantee
		err := f.db.View(operation.RetrieveGuarantee(collID, &guarantee))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve guarantee (%x): %w", collID, err)
		}
		guarantees = append(guarantees, &guarantee)
	}

	return guarantees, nil
}

func (f *Follower) Transactions(height uint64) ([]*flow.TransactionBody, error) {

	blockID, err := f.block(height)
	if errors.Is(err, dps.ErrTimeout) {
		return nil, dps.ErrTimeout
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var collIDs []flow.Identifier
	err = f.db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
	if err != nil {
		return nil, fmt.Errorf("could not lookup collections: %w", err)
	}

	var transactions []*flow.TransactionBody
	for _, collID := range collIDs {
		var collection flow.LightCollection
		err := f.db.View(operation.RetrieveCollection(collID, &collection))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve collection (%x): %w", collID, err)
		}
		for _, txID := range collection.Transactions {
			var transaction flow.TransactionBody
			err := f.db.View(operation.RetrieveTransaction(txID, &transaction))
			if err != nil {
				return nil, fmt.Errorf("could not retrieve transaction (%x): %w", txID, err)
			}
			transactions = append(transactions, &transaction)
		}
	}

	return transactions, nil
}

func (f *Follower) Results(height uint64) ([]*flow.TransactionResult, error) {
	blockID, err := f.block(height)
	if errors.Is(err, dps.ErrTimeout) {
		return nil, dps.ErrTimeout
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var results []flow.TransactionResult
	err = f.db.View(operation.LookupTransactionResultsByBlockID(blockID, &results))
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

func (f *Follower) Seals(height uint64) ([]*flow.Seal, error) {

	blockID, err := f.block(height)
	if errors.Is(err, dps.ErrTimeout) {
		return nil, dps.ErrTimeout
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	// LookupPayloadSeals() returns the IDs of all the seals in the specified block.
	// It should not be confused with LookupBlockSeal(), which returns the ID of the
	// *last* payload seal found in the block.
	var sealIDs []flow.Identifier
	err = f.db.View(operation.LookupPayloadSeals(blockID, &sealIDs))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve seal IDs: %w", err)
	}

	if len(sealIDs) == 0 {
		return nil, nil
	}

	seals := make([]*flow.Seal, 0, len(sealIDs))
	for _, s := range sealIDs {
		var seal flow.Seal
		err = f.db.View(operation.RetrieveSeal(s, &seal))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve seal: %w", err)
		}
		seals = append(seals, &seal)
	}

	return seals, nil
}

func (f *Follower) Events(height uint64) ([]flow.Event, error) {

	blockID, err := f.block(height)
	if errors.Is(err, dps.ErrTimeout) {
		return nil, dps.ErrTimeout
	}
	if err != nil {
		return nil, fmt.Errorf("could not get block for height: %w", err)
	}

	var events []flow.Event
	err = f.db.View(operation.LookupEventsByBlockID(blockID, &events))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	return events, nil
}

func (f *Follower) block(height uint64) (flow.Identifier, error) {

	if f.height == height {
		return f.blockID, nil
	}

	// If the requested height is not yet available, timeout until it is.
	if height > f.follower.Height() {
		return flow.Identifier{}, dps.ErrTimeout
	}

	// The protocol state maps everything by block ID. However, finalized blocks
	// are unambiguously available by height, so we can look up which block ID
	// corresponds to the desired height.
	var blockID flow.Identifier
	err := f.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not look up block: %w", err)
	}

	f.height = height
	f.blockID = blockID

	return blockID, nil
}
