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

package execution

import (
	"fmt"
	"math"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

type blockReader interface {
	Read(blockID flow.Identifier) ([]byte, error)
}

type Follower struct {
	log zerolog.Logger

	blocks blockReader
	codec  dps.Codec
	db     *badger.DB

	data   BlockData
	height uint64
	index int
}

func New(log zerolog.Logger, blocks blockReader, codec dps.Codec, db *badger.DB) *Follower {
	f := Follower{
		log: log,

		blocks: blocks,
		codec:  codec,
		db:     db,
		height: math.MaxUint64,
	}

	return &f
}

func (f *Follower) Height() uint64 {
	return f.height
}

func (f *Follower) Update() (*ledger.TrieUpdate, error) {
	if len(f.data.TrieUpdates) == 0 || f.index == len(f.data.TrieUpdates) {
		// No more trie updates are available.
		return nil, dps.ErrTimeout
	}

	// Copy next update to be returned.
	update := f.data.TrieUpdates[f.index]
	f.index++

	return update, nil
}

// Block returns the latest sealed block data.
func (f *Follower) Block() BlockData {
	return f.data
}

func (f *Follower) OnBlockFinalized(finalID flow.Identifier) {
	for {
		b, err := f.blocks.Read(finalID)
		if err != nil {
			// The block data is not yet available. Retry until it becomes available.
			time.Sleep(1 * time.Second)
		}

		var data BlockData
		err = f.codec.Unmarshal(b, &data)
		if err != nil {
			f.log.Error().Err(err).Msg("could not unmarshal block from execution follower")
			return
		}

		f.data = data
		f.height = data.Block.Header.Height
		f.index = 0

		break
	}

	f.IndexAll(finalID)
}

func (f *Follower) IndexAll(blockID flow.Identifier) {
	// FIXME: Sanity check with block seals and execution results.

	err := f.db.Update(func(txn *badger.Txn) error {
		var guarIDs []flow.Identifier
		for _, coll := range f.data.Collections {
			guarIDs = append(guarIDs, coll.Guarantee.ID())
			err := operation.InsertGuarantee(coll.Collection().ID(), coll.Guarantee)(txn)
			if err != nil {
				return fmt.Errorf("could not index guarantee: %w", err)
			}

			lightColl := coll.Collection().Light()
			err = operation.InsertCollection(&lightColl)(txn)
			if err != nil {
				return fmt.Errorf("could not index collection: %w", err)
			}
		}

		err := operation.IndexPayloadGuarantees(blockID, guarIDs)(txn)
		if err != nil {
			return fmt.Errorf("could not index payload guarantees: %w", err)
		}

		for _, result := range f.data.TxResults {
			err := operation.InsertTransactionResult(blockID, result)(txn)
			if err != nil {
				return fmt.Errorf("could not index transaction result: %w", err)
			}
		}

		var seals []flow.Identifier
		for _, seal := range f.data.Block.Payload.Seals {
			seals = append(seals, seal.ID())

			err := operation.InsertSeal(seal.ID(), seal)(txn)
			if err != nil {
				return fmt.Errorf("could not index seal: %w", err)
			}
		}
		err = operation.IndexPayloadSeals(blockID, seals)(txn)
		if err != nil {
			return fmt.Errorf("could not index payload seal: %w", err)
		}

		for _, event := range f.data.Events {
			err := operation.InsertEvent(blockID, *event)(txn)
			if err != nil {
				return fmt.Errorf("could not index event: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		f.log.Error().Err(err).Msg("could not index execution state")
	}
}
