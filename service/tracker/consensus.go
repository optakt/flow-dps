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

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

// Consensus is the DPS consensus follower, which uses a local protocol state
// database to retrieve consensus-dependent data, while falling back on an
// record holder to complement the rest of the data. It provides a callback for
// the unstaked consensus follower on the Flow network that allows it to update
// the cached data each time a block is finalized.
// Consensus implements the `Chain` interface needed by the DPS indexer.
type Consensus struct {
	log   zerolog.Logger
	chain dps.Chain
	hold  RecordHolder
	last  uint64
}

// NewConsensus returns a new instance of the DPS consensus follower, reading
// from the provided protocol state database and the provided block record
// holder.
func NewConsensus(log zerolog.Logger, chain dps.Chain, hold RecordHolder) (*Consensus, error) {

	c := Consensus{
		log:   log.With().Str("component", "consensus_tracker").Logger(),
		chain: chain,
		hold:  hold,
		last:  0,
	}

	return &c, nil
}

// Root returns the root height from the underlying protocol state.
func (c *Consensus) Root() (uint64, error) {
	root, err := c.chain.Root()
	c.last = root
	c.log.Debug().Uint64("height", root).Msg("root initialization processed")
	return root, err
}

// Height returns the height for the given block ID.
func (c *Consensus) Height(blockID flow.Identifier) (uint64, error) {
	return c.chain.Height(blockID)
}

// Header returns the header for the given height, if available. Once a header
// has been successfully retrieved, all block payload data at a height lower
// than the returned payload are purged from the cache.
func (c *Consensus) Header(height uint64) (*flow.Header, error) {
	if height > c.last {
		return nil, dps.ErrUnavailable
	}
	return c.chain.Header(height)
}

// Guarantees returns the collection guarantees for the given height, if available.
func (c *Consensus) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {
	if height > c.last {
		return nil, dps.ErrUnavailable
	}
	return c.chain.Guarantees(height)
}

// Seals returns the block seals for the given height, if available.
func (c *Consensus) Seals(height uint64) ([]*flow.Seal, error) {
	if height > c.last {
		return nil, dps.ErrUnavailable
	}
	return c.chain.Seals(height)
}

// Commit returns the state commitment for the given height, if available.
func (c *Consensus) Commit(height uint64) (flow.StateCommitment, error) {
	header, err := c.Header(height)
	if err != nil {
		return flow.DummyStateCommitment, fmt.Errorf("could not get header for height: %w", err)
	}
	record, ok := c.hold.Record(header.ID())
	if !ok {
		return flow.DummyStateCommitment, dps.ErrUnavailable
	}
	return record.FinalStateCommitment, nil
}

// Collections returns the light collections for the finalized block at the
// given height.
func (c *Consensus) Collections(height uint64) ([]*flow.LightCollection, error) {
	header, err := c.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header for height: %w", err)
	}
	record, ok := c.hold.Record(header.ID())
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
	header, err := c.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header for height: %w", err)
	}
	record, ok := c.hold.Record(header.ID())
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
	header, err := c.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header for height: %w", err)
	}
	record, ok := c.hold.Record(header.ID())
	if !ok {
		return nil, dps.ErrUnavailable
	}
	return record.TxResults, nil
}

// Events returns the transaction events for the finalized block at the
// given height.
func (c *Consensus) Events(height uint64) ([]flow.Event, error) {
	header, err := c.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header for height: %w", err)
	}
	record, ok := c.hold.Record(header.ID())
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
	height, err := c.Height(finalID)
	if err != nil {
		c.log.Error().Err(err).Hex("block", finalID[:]).Msg("could not get height for block")
		return
	}
	c.log.Debug().Hex("block", finalID[:]).Uint64("height", height).Msg("block finalization processed")
	c.last = height
}
