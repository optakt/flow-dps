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

package consensus

import (
	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"
)

// Follower is a wrapper around the database that the Flow consensus follower populates. It is used to
// expose the current height and block ID of the consensus follower's last finalized block.
type Follower struct {
	log zerolog.Logger

	db *badger.DB

	height  uint64
	blockID flow.Identifier
}

// New returns a new Follower instance.
func New(log zerolog.Logger, db *badger.DB) *Follower {
	f := Follower{
		log: log,
		db:  db,
	}

	return &f
}

// OnBlockFinalized is a callback that is used to update the state of the Follower.
func (f *Follower) OnBlockFinalized(finalID flow.Identifier) {
	var height uint64
	err := f.db.View(operation.RetrieveFinalizedHeight(&height))
	if err != nil {
		f.log.Error().Err(err).Msg("Could not retrieve finalized block height")
		return
	}

	f.height = height
	f.blockID = finalID
}

// Height returns the last finalized height according to the consensus follower.
func (f *Follower) Height() uint64 {
	return f.height
}

// BlockID returns the last finalized block's ID according to the consensus follower.
func (f *Follower) BlockID() flow.Identifier {
	return f.blockID
}
