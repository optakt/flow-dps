// Copyright 2021 Alvalor S.A.
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

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/awfm9/flow-dps/model/dps"
)

type ProtocolState struct {
	db *badger.DB
}

func FromProtocolState(dir string) (*ProtocolState, error) {

	opts := badger.DefaultOptions(dir).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not open badger database: %w", err)
	}

	ps := ProtocolState{
		db: db,
	}

	return &ps, nil
}

func (ps *ProtocolState) Header(height uint64) (*flow.Header, error) {
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(ps.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}
	if err != nil {
		return nil, fmt.Errorf("could not look up block: %w", err)
	}
	var header flow.Header
	err = operation.RetrieveHeader(blockID, &header)(ps.db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve header: %w", err)
	}
	return &header, nil
}

func (ps *ProtocolState) Commit(height uint64) (flow.StateCommitment, error) {
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(ps.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}
	var sealID flow.Identifier
	err = operation.LookupBlockSeal(blockID, &sealID)(ps.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}
	if err != nil {
		return nil, fmt.Errorf("could not look up seal: %w", err)
	}
	var seal flow.Seal
	err = operation.RetrieveSeal(sealID, &seal)(ps.db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve seal: %w", err)
	}
	return seal.FinalState, nil
}

func (ps *ProtocolState) Events(height uint64) ([]flow.Event, error) {
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(ps.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, dps.ErrFinished
	}
	var events []flow.Event
	err = operation.LookupEventsByBlockID(blockID, &events)(ps.db.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	return events, nil
}
