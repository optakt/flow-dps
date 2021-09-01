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

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

type Source struct {
	log zerolog.Logger

	db        *badger.DB
	execution ExecutionFollower
	consensus ConsensusFollower

	blockID flow.Identifier
}

func NewSource(log zerolog.Logger, execution ExecutionFollower, consensus ConsensusFollower, db *badger.DB) *Source {
	s := Source{
		log: log,

		db:        db,
		execution: execution,
		consensus: consensus,

		blockID: flow.ZeroID,
	}

	return &s
}

func (s *Source) Update() (*ledger.TrieUpdate, error) {
	update, err := s.execution.Update()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve trie update: %w", err)
	}
	return update, nil
}

func (s *Source) Root() (uint64, error) {
	var height uint64
	err := s.db.View(operation.RetrieveRootHeight(&height))
	if err != nil {
		return 0, fmt.Errorf("could not look up root height: %w", err)
	}
	return height, nil
}

func (s *Source) Commit(height uint64) (flow.StateCommitment, error) {
	if height > s.consensus.Height() {
		return flow.StateCommitment{}, dps.ErrUnavailable
	}

	blockID := s.consensus.BlockID()

	var commit flow.StateCommitment
	err := s.db.View(operation.LookupStateCommitment(blockID, &commit))
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not look up commit: %w", err)
	}

	return commit, nil
}

func (s *Source) Header(height uint64) (*flow.Header, error) {
	header, err := s.execution.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve header: %w", err)
	}

	return header, nil
}

func (s *Source) Collections(height uint64) ([]*flow.LightCollection, error) {
	collections, err := s.execution.Collections(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve collections: %w", err)
	}

	return collections, nil
}

func (s *Source) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {
	guarantees, err := s.execution.Guarantees(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve guarantees: %w", err)
	}

	return guarantees, nil
}

func (s *Source) Transactions(height uint64) ([]*flow.TransactionBody, error) {
	transactions, err := s.execution.Transactions(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transactions: %w", err)
	}

	return transactions, nil
}

func (s *Source) Results(height uint64) ([]*flow.TransactionResult, error) {
	txResults, err := s.execution.Results(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transaction results: %w", err)
	}

	return txResults, nil
}

func (s *Source) Seals(height uint64) ([]*flow.Seal, error) {
	seals, err := s.execution.Seals(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve seals: %w", err)
	}

	return seals, nil
}

func (s *Source) Events(height uint64) ([]flow.Event, error) {
	events, err := s.execution.Events(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	return events, nil
}
