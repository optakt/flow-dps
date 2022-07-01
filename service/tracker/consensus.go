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

package tracker

import (
	"fmt"
	"github.com/onflow/flow-go/consensus/hotstuff/model"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/onflow/flow-dps/models/dps"
)

// Consensus is the DPS consensus follower, which uses a local protocol state
// database to retrieve consensus-dependent data, while falling back on an
// record holder to complement the rest of the data. It provides a callback for
// the unstaked consensus follower on the Flow network that allows it to update
// the cached data each time a block is finalized.
// Consensus implements the `Chain` interface needed by the DPS indexer.
type Consensus struct {
	log  zerolog.Logger
	db   *badger.DB
	hold RecordHolder
	last uint64
}

// NewConsensus returns a new instance of the DPS consensus follower, reading
// from the provided protocol state database and the provided block record
// holder.
func NewConsensus(log zerolog.Logger, db *badger.DB, hold RecordHolder) (*Consensus, error) {

	var last uint64
	err := db.View(operation.RetrieveFinalizedHeight(&last))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last height: %w", err)
	}

	c := Consensus{
		log:  log.With().Str("component", "consensus_tracker").Logger(),
		db:   db,
		hold: hold,
		last: last,
	}

	return &c, nil
}

// OnBlockFinalized is a callback that notifies the consensus tracker of a new
// finalized block.
func (c *Consensus) OnBlockFinalized(block *model.Block) {
	blockID := block.BlockID

	var header flow.Header
	err := c.db.View(operation.RetrieveHeader(blockID, &header))
	if err != nil {
		c.log.Error().Err(err).Hex("block", blockID[:]).Msg("could not get header")
		return
	}
	c.last = header.Height
	c.log.Debug().Hex("block", blockID[:]).Uint64("height", header.Height).Msg("block finalization processed")
}

// Root returns the root height from the underlying protocol state.
func (c *Consensus) Root() (uint64, error) {

	var root uint64
	err := c.db.View(operation.RetrieveRootHeight(&root))
	if err != nil {
		return 0, fmt.Errorf("could not retrieve root height: %w", err)
	}

	return root, nil
}

// Header returns the header for the given height, if available. Once a header
// has been successfully retrieved, all block payload data at a height lower
// than the returned payload are purged from the cache.
func (c *Consensus) Header(height uint64) (*flow.Header, error) {

	if height > c.last {
		return nil, dps.ErrUnavailable
	}

	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}

	var header flow.Header
	err = c.db.View(operation.RetrieveHeader(blockID, &header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve header: %w", err)
	}

	return &header, nil
}

// Guarantees returns the collection guarantees for the given height, if available.
func (c *Consensus) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {

	if height > c.last {
		return nil, dps.ErrUnavailable
	}

	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}

	var collIDs []flow.Identifier
	err = c.db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
	if err != nil {
		return nil, fmt.Errorf("could not lookup collections: %w", err)
	}

	guarantees := make([]*flow.CollectionGuarantee, 0, len(collIDs))
	for _, collID := range collIDs {
		var guarantee flow.CollectionGuarantee
		err := c.db.View(operation.RetrieveGuarantee(collID, &guarantee))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve guarantee (%x): %w", collID, err)
		}
		guarantees = append(guarantees, &guarantee)
	}

	return guarantees, nil
}

// Seals returns the block seals for the given height, if available.
func (c *Consensus) Seals(height uint64) ([]*flow.Seal, error) {

	if height > c.last {
		return nil, dps.ErrUnavailable
	}

	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}

	var sealIDs []flow.Identifier
	err = c.db.View(operation.LookupPayloadSeals(blockID, &sealIDs))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve seal IDs: %w", err)
	}

	if len(sealIDs) == 0 {
		return nil, nil
	}

	seals := make([]*flow.Seal, 0, len(sealIDs))
	for _, sealID := range sealIDs {
		var seal flow.Seal
		err = c.db.View(operation.RetrieveSeal(sealID, &seal))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve seal: %w", err)
		}
		seals = append(seals, &seal)
	}

	return seals, nil
}

// Commit returns the state commitment for the given height, if available.
func (c *Consensus) Commit(height uint64) (flow.StateCommitment, error) {

	if height > c.last {
		return flow.DummyStateCommitment, dps.ErrUnavailable
	}

	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return flow.DummyStateCommitment, fmt.Errorf("could not look up block: %w", err)
	}

	record, err := c.hold.Record(blockID)
	if err != nil {
		return flow.DummyStateCommitment, fmt.Errorf("could not get record: %w", err)
	}

	return record.FinalStateCommitment, nil
}

// Collections returns the light collections for the finalized block at the
// given height.
func (c *Consensus) Collections(height uint64) ([]*flow.LightCollection, error) {

	if height > c.last {
		return nil, dps.ErrUnavailable
	}

	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}

	record, err := c.hold.Record(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get record: %w", err)
	}

	collections := make([]*flow.LightCollection, 0, len(record.Collections))
	for _, complete := range record.Collections {
		collection := complete.Collection().Light()
		collections = append(collections, &collection)
	}

	return collections, nil
}

// Transactions returns the transaction bodies for the finalized block at the
// given height.
func (c *Consensus) Transactions(height uint64) ([]*flow.TransactionBody, error) {

	if height > c.last {
		return nil, dps.ErrUnavailable
	}

	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}

	record, err := c.hold.Record(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get record: %w", err)
	}

	transactions := make([]*flow.TransactionBody, 0, len(record.Collections))
	for _, complete := range record.Collections {
		transactions = append(transactions, complete.Transactions...)
	}

	return transactions, nil
}

// Results returns the transaction results for the finalized block at the
// given height.
func (c *Consensus) Results(height uint64) ([]*flow.TransactionResult, error) {

	if height > c.last {
		return nil, dps.ErrUnavailable
	}

	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}

	record, err := c.hold.Record(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get record: %w", err)
	}

	return record.TxResults, nil
}

// Events returns the transaction events for the finalized block at the
// given height.
func (c *Consensus) Events(height uint64) ([]flow.Event, error) {

	if height > c.last {
		return nil, dps.ErrUnavailable
	}

	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}

	record, err := c.hold.Record(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get record: %w", err)
	}

	events := make([]flow.Event, 0, len(record.Events))
	for _, event := range record.Events {
		events = append(events, *event)
	}

	return events, nil
}
