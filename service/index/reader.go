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

package index

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/service/storage"
)

type Reader struct {
	db *badger.DB
}

func NewReader(db *badger.DB) *Reader {

	r := Reader{
		db: db,
	}

	return &r
}

func (r *Reader) Last() (uint64, error) {
	var height uint64
	err := r.db.View(storage.RetrieveLastHeight(&height))
	return height, err
}

func (r *Reader) Header(height uint64) (*flow.Header, error) {
	var header flow.Header
	err := r.db.View(storage.RetrieveHeader(height, &header))
	return &header, err
}

func (r *Reader) Commit(height uint64) (flow.StateCommitment, error) {
	var commit flow.StateCommitment
	err := r.db.View(storage.RetrieveCommitByHeight(height, &commit))
	return commit, err
}

func (r *Reader) Events(height uint64, types ...flow.EventType) ([]flow.Event, error) {
	// TODO: Introduce a height check here that doesn't need to know about
	// current indexing progress.
	var events []flow.Event
	err := r.db.View(storage.RetrieveEvents(height, types, &events))
	return events, err
}

func (r *Reader) Registers(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
	// TODO: Introduce a height check here that doesn't need to know about
	// current indexing progress.
	values := make([]ledger.Value, 0, len(paths))
	err := r.db.View(func(tx *badger.Txn) error {
		for _, path := range paths {
			var payload ledger.Payload
			err := storage.RetrievePayload(height, path, &payload)(tx)
			if err != nil {
				return fmt.Errorf("could not retrieve payload (path: %x): %w", path, err)
			}
			values = append(values, payload.Value)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return values, nil
}
