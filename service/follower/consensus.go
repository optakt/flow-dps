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

package follower

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

// Consensus is a wrapper around the database that the Flow consensus follower populates. It is used to
// expose the current height and block ID of the consensus follower's last finalized block.
type Consensus struct {
	log      zerolog.Logger
	db       *badger.DB
	hold     RecordHolder
	payloads map[uint64]*Payload
}

// NewConsensus returns a new Consensus instance.
func NewConsensus(log zerolog.Logger, db *badger.DB, hold RecordHolder) *Consensus {
	f := Consensus{
		log:      log,
		db:       db,
		hold:     hold,
		payloads: make(map[uint64]*Payload),
	}

	return &f
}

// Root returns the root height from the underlying protocol state.
func (c *Consensus) Root() (uint64, error) {
	var height uint64
	err := c.db.View(operation.RetrieveRootHeight(&height))
	if err != nil {
		return 0, fmt.Errorf("could not retrieve root height: %w", err)
	}
	return height, nil
}

// Header returns the header for the given height, if available.
func (c *Consensus) Header(height uint64) (*flow.Header, error) {
	c.purge(height)
	payload, ok := c.payloads[height]
	if !ok {
		return nil, dps.ErrUnavailable
	}
	return payload.Header, nil
}

// Guarantees returns the collection guarantees for the given height, if available.
func (c *Consensus) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {
	payload, ok := c.payloads[height]
	if !ok {
		return nil, dps.ErrUnavailable
	}
	return payload.Guarantees, nil
}

// Seals returns the block seals for the given height, if available.
func (c *Consensus) Seals(height uint64) ([]*flow.Seal, error) {
	payload, ok := c.payloads[height]
	if !ok {
		return nil, dps.ErrUnavailable
	}
	return payload.Seals, nil
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
	return record.Commit, nil
}

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

// OnBlockFinalized is a callback that is used to update the state of the Consensus.
func (c *Consensus) OnBlockFinalized(finalID flow.Identifier) {
	err := c.indexPayload(finalID)
	if err != nil {
		c.log.Error().Err(err).Msg("could not index block payload")
	}
}

func (c *Consensus) indexPayload(finalID flow.Identifier) error {

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
