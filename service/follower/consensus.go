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
	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"
)

// Consensus is a wrapper around the database that the Flow consensus follower populates. It is used to
// expose the current height and block ID of the consensus follower's last finalized block.
type Consensus struct {
	log zerolog.Logger

	db        *badger.DB
	execution *Execution
}

// NewConsensus returns a new Consensus instance.
func NewConsensus(log zerolog.Logger, db *badger.DB, execution *Execution) *Consensus {
	f := Consensus{
		log: log,
		db:  db,
	}

	return &f
}

// OnBlockFinalized is a callback that is used to update the state of the Consensus.
func (c *Consensus) OnBlockFinalized(finalID flow.Identifier) {

	// Here, we should preload everything so we can index more efficiently.

}
