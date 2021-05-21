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

package state

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/service/storage"
)

type Index struct {
	core *Core
}

// TODO: check if there is an intermediate representation of Flow block headers
// that contains everything we need for the access and Rosetta APIs, but drops
// a lot of superfluous data (i.e. maybe signatures?)
// => https://github.com/awfm9/flow-dps/issues/39

func (i *Index) Header(height uint64, header *flow.Header) error {
	err := i.core.db.Update(func(tx *badger.Txn) error {

		// use the headers height as key to store the encoded header
		err := storage.SaveHeaderForHeight(height, header)(tx)
		if err != nil {
			return fmt.Errorf("could not persist header data: %w", err)
		}

		// create an index to map block ID to height
		err = storage.SaveHeightForBlock(header.ID(), height)(tx)
		if err != nil {
			return fmt.Errorf("could not persist block index: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not index header: %w", err)
	}
	return nil
}

func (i *Index) Commit(height uint64, commit flow.StateCommitment) error {
	err := i.core.db.Update(func(tx *badger.Txn) error {

		// create an index to map commit to height
		err := storage.SaveCommitForHeight(commit, height)(tx)
		if err != nil {
			return fmt.Errorf("could not persist commit index: %w", err)
		}

		// create an index to map height to commit
		err = storage.SaveHeightForCommit(height, commit)(tx)
		if err != nil {
			return fmt.Errorf("could not persist height index: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not index commit: %w", err)
	}
	return nil
}

func (i *Index) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	return i.core.db.Update(func(tx *badger.Txn) error {
		for i, path := range paths {
			payload := payloads[i]
			err := storage.SavePayload(height, path, payload)(tx)
			if err != nil {
				return fmt.Errorf("could not store payload (path: %x): %w", path, err)
			}
		}
		return nil
	})
}

func (i *Index) Events(height uint64, events []flow.Event) error {
	err := i.core.db.Update(func(tx *badger.Txn) error {

		buckets := make(map[flow.EventType][]flow.Event)
		for _, event := range events {
			buckets[event.Type] = append(buckets[event.Type], event)
		}

		for typ, evts := range buckets {
			err := storage.SaveEvents(height, typ, evts)(tx)
			if err != nil {
				return fmt.Errorf("could not persist events: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not index events: %w", err)
	}

	return nil
}

func (i *Index) Last(commit flow.StateCommitment) error {
	err := i.core.db.Update(storage.SaveLastCommit(commit))
	return err
}
