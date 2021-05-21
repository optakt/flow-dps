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
	"github.com/dgraph-io/badger/v2/options"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/models/dps"
)

type ProtocolState struct {
	db *badger.DB
}

func DefaultOptions(dir string) badger.Options {
	return badger.DefaultOptions(dir).
		WithMaxTableSize(256 << 20).
		WithValueLogFileSize(64 << 20).
		WithTableLoadingMode(options.FileIO).
		WithValueLogLoadingMode(options.FileIO).
		WithNumMemtables(1).
		WithKeepL0InMemory(false).
		WithCompactL0OnClose(false).
		WithNumLevelZeroTables(1).
		WithNumLevelZeroTablesStall(2).
		WithLoadBloomsOnOpen(false).
		WithIndexCacheSize(2000 << 20).
		WithBlockCacheSize(0).
		WithLogger(nil)
}

func FromProtocolState(db *badger.DB) *ProtocolState {
	ps := ProtocolState{
		db: db,
	}

	return &ps
}

func (ps *ProtocolState) Root() (uint64, error) {
	var height uint64
	err := operation.RetrieveRootHeight(&height)(ps.db.NewTransaction(false))
	if err != nil {
		return 0, fmt.Errorf("could not look up root height: %w", err)
	}
	return height, nil
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
		return flow.StateCommitment{}, dps.ErrFinished
	}
	var sealID flow.Identifier
	err = operation.LookupBlockSeal(blockID, &sealID)(ps.db.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return flow.StateCommitment{}, dps.ErrFinished
	}
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not look up seal: %w", err)
	}
	var seal flow.Seal
	err = operation.RetrieveSeal(sealID, &seal)(ps.db.NewTransaction(false))
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not retrieve seal: %w", err)
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
