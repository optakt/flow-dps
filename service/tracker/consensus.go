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
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

// Consensus is the DPS consensus follower, which uses a local protocol state
// database to retrieve consensus-dependent data, while falling back on an
// record holder to complement the rest of the data. It provides a callback for
// the unstaked consensus follower on the Flow network that allows it to update
// the cached data each time a block is finalized.
// Consensus implements the `Chain` interface needed by the DPS indexer.
type Consensus struct {
	log      zerolog.Logger
	db       *badger.DB
	hold     RecordHolder
	last     uint64
	payloads map[uint64]*Payload
}

// NewConsensus returns a new instance of the DPS consensus follower, reading
// from the provided protocol state database and the provided block record
// holder.
func NewConsensus(log zerolog.Logger, db *badger.DB, hold RecordHolder) (*Consensus, error) {

	// We first try to retrieve the root height from the database, in case there
	// is already protocol state on the disk. We then keep track of the last
	// finalized height in order to determine which requests can be fulfilled
	// from disk.
	last := uint64(0)
	err := db.View(operation.RetrieveFinalizedHeight(&last))
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, fmt.Errorf("could not retrieve finalized height: %w", err)
	}

	c := Consensus{
		log:      log.With().Str("component", "consensus_tracker").Logger(),
		db:       db,
		hold:     hold,
		payloads: make(map[uint64]*Payload),
		last:     last,
	}

	return &c, nil
}

// Root returns the root height from the underlying protocol state.
func (c *Consensus) Root() (uint64, error) {

	var root uint64
	err := c.db.View(operation.RetrieveRootHeight(&root))
	if err != nil {
		return 0, fmt.Errorf("could not retrieve root height: %w", err)
	}

	// FIXME: This is currently a "hack" so that we set the last height to root
	// height when starting. It is needed because the protocol state is
	// boostrapped by the unstaked consensus follower; this means that there is
	// no root height available in the database when we initialize the DPS
	// consensus follower. This should be addressed by properly handling startup
	// order and waiting, but this is a quick fix that will work for testing.
	c.last = root

	return root, nil
}

// Header returns the header for the given height, if available. Once a header
// has been successfully retrieved, all block payload data at a height lower
// than the returned payload are purged from the cache.
func (c *Consensus) Header(height uint64) (*flow.Header, error) {

	// Once a height is requested, all data from previous heights becomes
	// irrelevant. This means we can purge the cache with the new height as a
	// threshold. We only need to do that in one method, so `Header` will do.
	c.purge(height)

	// If we have the payload cached, we can return the header immediately. The
	// same logic applies to all other functions.
	payload, ok := c.payloads[height]
	if ok {
		return payload.Header, nil
	}

	// If the payload is not cached and the last finalized height is behind the
	// requested height, we have not received the requested data yet and it is
	// thus unavailable. We are probably following consensus in real-time now.
	if height > c.last {
		return nil, dps.ErrUnavailable
	}

	// Otherwise, we should have the header in the on-disk protocol state
	// database. This can happen for the root block, or in cases where we start
	// with an existing on-disk protocol state.
	// FIXME: Remove redundancy of retrieval code.
	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}
	var header flow.Header
	err = c.db.View(operation.RetrieveHeader(blockID, &header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve block: %w", err)
	}

	return &header, nil
}

// Guarantees returns the collection guarantees for the given height, if available.
func (c *Consensus) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {

	payload, ok := c.payloads[height]
	if ok {
		return payload.Guarantees, nil
	}

	// FIXME: Remove redundancy of retrieval code.
	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}
	var collIDs []flow.Identifier
	err = c.db.View(operation.LookupPayloadGuarantees(blockID, &collIDs))
	if err != nil {
		return nil, fmt.Errorf("could not look up guarantees: %w", err)
	}
	guarantees := make([]*flow.CollectionGuarantee, 0, len(collIDs))
	for _, collID := range collIDs {
		var guarantee flow.CollectionGuarantee
		err = c.db.View(operation.RetrieveGuarantee(collID, &guarantee))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve guarantee (collection: %x): %w", collID, err)
		}
		guarantees = append(guarantees, &guarantee)
	}

	return guarantees, nil
}

// Seals returns the block seals for the given height, if available.
func (c *Consensus) Seals(height uint64) ([]*flow.Seal, error) {

	payload, ok := c.payloads[height]
	if ok {
		return payload.Seals, nil
	}

	// FIXME: Remove redundancy of retrieval code.
	var blockID flow.Identifier
	err := c.db.View(operation.LookupBlockHeight(height, &blockID))
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}
	var sealIDs []flow.Identifier
	err = c.db.View(operation.LookupPayloadSeals(blockID, &sealIDs))
	if err != nil {
		return nil, fmt.Errorf("could not look up seals: %w", err)
	}
	seals := make([]*flow.Seal, 0, len(sealIDs))
	for _, sealID := range sealIDs {
		var seal flow.Seal
		err = c.db.View(operation.RetrieveSeal(sealID, &seal))
		if err != nil {
			return nil, fmt.Errorf("could not retrieve seal (seal: %x): %w", sealID, err)
		}
		seals = append(seals, &seal)
	}

	return seals, nil
}

// Commit returns the state commitment for the given height, if available.
func (c *Consensus) Commit(height uint64) (flow.StateCommitment, error) {
	payload, ok := c.payloads[height]
	if !ok {
		return flow.DummyStateCommitment, dps.ErrUnavailable
	}
	record, ok := c.hold.Record(payload.Header.ID())
	if !ok {
		return flow.DummyStateCommitment, dps.ErrUnavailable
	}
	return record.FinalStateCommitment, nil
}

// Collections returns the light collections for the finalized block at the
// given height.
func (c *Consensus) Collections(height uint64) ([]*flow.LightCollection, error) {
	payload, ok := c.payloads[height]
	if !ok {
		return nil, dps.ErrUnavailable
	}
	record, ok := c.hold.Record(payload.Header.ID())
	if !ok {
		return nil, dps.ErrUnavailable
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
	payload, ok := c.payloads[height]
	if !ok {
		return nil, dps.ErrUnavailable
	}
	record, ok := c.hold.Record(payload.Header.ID())
	if !ok {
		return nil, dps.ErrUnavailable
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
	payload, ok := c.payloads[height]
	if !ok {
		return nil, dps.ErrUnavailable
	}
	record, ok := c.hold.Record(payload.Header.ID())
	if !ok {
		return nil, dps.ErrUnavailable
	}
	return record.TxResults, nil
}

// Events returns the transaction events for the finalized block at the
// given height.
func (c *Consensus) Events(height uint64) ([]flow.Event, error) {
	payload, ok := c.payloads[height]
	if !ok {
		return nil, dps.ErrUnavailable
	}
	record, ok := c.hold.Record(payload.Header.ID())
	if !ok {
		return nil, dps.ErrUnavailable
	}
	events := make([]flow.Event, 0, len(record.Events))
	for _, event := range record.Events {
		events = append(events, *event)
	}
	return events, nil
}

// OnBlockFinalized is a callback that notifies the consensus tracker of a new
// finalized block.
func (c *Consensus) OnBlockFinalized(finalID flow.Identifier) {

	log := c.log.With().Hex("block", finalID[:]).Logger()

	err := c.processPayload(finalID)
	if err != nil {
		c.log.Error().Err(err).Msg("could not index block payload")
		return
	}

	log.Debug().Msg("finalized block payload processed")
}

func (c *Consensus) processPayload(finalID flow.Identifier) error {

	var header flow.Header
	var guarantees []*flow.CollectionGuarantee
	var seals []*flow.Seal
	err := c.db.View(func(tx *badger.Txn) error {

		err := operation.RetrieveHeader(finalID, &header)(tx)
		if err != nil {
			return fmt.Errorf("could not retrieve header: %w", err)
		}

		var collIDs []flow.Identifier
		err = operation.LookupPayloadGuarantees(finalID, &collIDs)(tx)
		if err != nil {
			return fmt.Errorf("could not look up guarantees: %w", err)
		}

		guarantees = make([]*flow.CollectionGuarantee, 0, len(collIDs))
		for _, collID := range collIDs {
			var guarantee flow.CollectionGuarantee
			err = operation.RetrieveGuarantee(collID, &guarantee)(tx)
			if err != nil {
				return fmt.Errorf("could not retrieve guarantee (collection: %x): %w", collID, err)
			}
			guarantees = append(guarantees, &guarantee)
		}

		var sealIDs []flow.Identifier
		err = operation.LookupPayloadSeals(finalID, &sealIDs)(tx)
		if err != nil {
			return fmt.Errorf("could not look up seals: %w", err)
		}

		seals = make([]*flow.Seal, 0, len(sealIDs))
		for _, sealID := range sealIDs {
			var seal flow.Seal
			err = operation.RetrieveSeal(sealID, &seal)(tx)
			if err != nil {
				return fmt.Errorf("could not retrieve seal: %w", err)
			}
			seals = append(seals, &seal)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not retrieve payload data: %w", err)
	}

	// TODO: Do a sanity check of the sealed state commitment from each seal
	// against the state commitment we already stored from the execution data
	// for that block:
	// => https://github.com/optakt/flow-dps/issues/395

	payload := Payload{
		Header:     &header,
		Guarantees: guarantees,
		Seals:      seals,
	}

	c.payloads[header.Height] = &payload

	return nil
}

func (c *Consensus) purge(threshold uint64) {
	for height := range c.payloads {
		if height < threshold {
			delete(c.payloads, height)
		}
	}
}
