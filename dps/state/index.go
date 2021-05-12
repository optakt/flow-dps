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
	"encoding/binary"
	"fmt"

	"github.com/OneOfOne/xxhash"
	"github.com/awfm9/flow-dps/models/dps"
	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/model/flow"
)

type Index struct {
	core *Core
}

func (i *Index) Header(height uint64, header *flow.Header) error {
	err := i.core.db.Update(func(tx *badger.Txn) error {

		// use the headers height as key to store the encoded header
		key := make([]byte, 1+8)
		key[0] = prefixHeaderData
		binary.BigEndian.PutUint64(key[1:1+8], height)
		val, err := cbor.Marshal(header, cbor.CanonicalEncOptions())
		if err != nil {
			return fmt.Errorf("could not encode header: %w", err)
		}
		val = i.core.compressor.EncodeAll(val, nil)
		err = tx.Set(key, val)
		if err != nil {
			return fmt.Errorf("could not save header data: %w", err)
		}

		// create an index to map block ID to height
		blockID := header.ID()
		key = make([]byte, 1+len(blockID))
		key[0] = prefixBlockIndex
		copy(key[1:], blockID[:])
		val = make([]byte, 8)
		binary.BigEndian.PutUint64(val[0:8], height)
		err = tx.Set(key, val)
		if err != nil {
			return fmt.Errorf("could not save block index: %w", err)
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
		key := make([]byte, 1+len(commit))
		key[0] = prefixCommitIndex
		copy(key[1:], commit)
		val := make([]byte, 8)
		binary.BigEndian.PutUint64(val[0:8], height)
		err := tx.Set(key, val)
		if err != nil {
			return fmt.Errorf("could not save commit index: %w", err)
		}

		// create an index to map height to commit
		key = make([]byte, 1+8)
		key[0] = prefixHeightIndex
		binary.BigEndian.PutUint64(key[1:1+8], height)
		err = tx.Set(key, commit)
		if err != nil {
			return fmt.Errorf("could not save height index: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not index commit: %w", err)
	}
	return nil
}

func (i *Index) Deltas(height uint64, deltas []dps.Delta) error {

	// finally, we index the payload for every path that has changed in this block
	err := i.core.db.Update(func(tx *badger.Txn) error {
		for _, delta := range deltas {
			for _, change := range delta {
				key := make([]byte, 1+pathfinder.PathByteSize+8)
				key[0] = prefixDeltaData
				copy(key[1:1+pathfinder.PathByteSize], change.Path)
				binary.BigEndian.PutUint64(key[1+pathfinder.PathByteSize:], height)
				val, err := cbor.Marshal(change.Payload, cbor.CanonicalEncOptions())
				if err != nil {
					return fmt.Errorf("could not encode delta: %w", err)
				}
				val = i.core.compressor.EncodeAll(val, nil)
				err = tx.Set(key, val)
				if err != nil {
					return fmt.Errorf("could not save delta data: %w", err)
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
			// Prefix + Block Height + Type Hash
			key := make([]byte, 1+8+8)
			key[0] = prefixEventData
			binary.BigEndian.PutUint64(key[1:1+8], height)
			binary.BigEndian.PutUint64(key[1+8:1+8+8], hash)

			val, err := cbor.Marshal(evts, cbor.CanonicalEncOptions())
			if err != nil {
				return fmt.Errorf("could not encode events: %w", err)
			}
			val = i.core.compressor.EncodeAll(val, nil)
			err = tx.Set(key, val)
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
		key := []byte{prefixLastCommit}
		err := tx.Set(key, commit)
		if err != nil {
			return fmt.Errorf("could not save last commit: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("could not index last commit: %w", err)
	}
	return nil
}
