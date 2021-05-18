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
	"errors"
	"fmt"

	"github.com/OneOfOne/xxhash"
	"github.com/awfm9/flow-dps/service/state/storage"
	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
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
		blockID := header.ID()
		err = storage.SaveHeightForBlock(blockID[:], height)(tx)
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

func (i *Index) Deltas(height uint64, deltas []dps.Delta) error {

	err := i.core.db.Update(func(tx *badger.Txn) error {
		for _, delta := range deltas {
			for _, change := range delta {
				err := storage.SaveDeltasForHeight(height, change)(tx)
				if err != nil {
					return fmt.Errorf("could not persist delta data: %w", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("could not index deltas: %w", err)
	}

	return nil
}

func (i *Index) Events(height uint64, events []flow.Event) error {
	err := i.core.db.Update(func(tx *badger.Txn) error {

		buckets := make(map[uint64][]flow.Event)
		for _, event := range events {
			hash := xxhash.Checksum64([]byte(event.Type))
			buckets[hash] = append(buckets[hash], event)
		}

		for hash, evts := range buckets {
			err := storage.SaveEventsForHeight(height, hash, evts)(tx)
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
	err := i.core.db.Update(func(tx *badger.Txn) error {
		err := storage.SaveLastCommit(commit)(tx)
		if err != nil {
			return fmt.Errorf("could not persist last commit: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("could not index last commit: %w", err)
	}
	return nil
}

func (i *Index) Compact() error {

	err := i.core.db.Sync()
	if err != nil {
		return fmt.Errorf("could not sync database: %w", err)
	}

	err = i.core.db.Flatten(4)
	if err != nil {
		return fmt.Errorf("could not flatten database: %w", err)
	}

	err = i.core.db.RunValueLogGC(0.5)
	if errors.Is(err, badger.ErrNoRewrite) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not run value log garbage collection: %w", err)
	}

	return nil
}
