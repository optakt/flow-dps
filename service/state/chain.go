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
	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/flow-go/model/flow"
)

type Chain struct {
	core *Core
}

func (c *Chain) Header(height uint64) (*flow.Header, error) {
	var header flow.Header

	err := c.core.db.View(func(tx *badger.Txn) error {
		key := make([]byte, 1+8)
		key[0] = prefixDataHeader
		binary.BigEndian.PutUint64(key[1:1+8], height)

		item, err := tx.Get(key)
		if err != nil {
			return fmt.Errorf("could not retrieve header at height %d: %w", height, err)
		}

		err = item.Value(func(val []byte) error {
			val, err := c.core.decompressor.DecodeAll(val, nil)
			if err != nil {
				return fmt.Errorf("could not decompress header: %w", err)
			}
			err = cbor.Unmarshal(val, &header)
			if err != nil {
				return fmt.Errorf("could not decode header: %w", err)
			}
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})

	return &header, err
}

func (c *Chain) Events(height uint64, types ...string) ([]flow.Event, error) {
	// Make sure that the request is for a height below the currently active
	// sentinel height; otherwise, we haven't indexed yet and we might return
	// false information.
	if height > c.core.height {
		return nil, fmt.Errorf("unknown height (current: %d, requested: %d)", c.core.height, height)
	}

	// FIXME: Should we keep the types filtering or not?
	lookup := make(map[uint64]struct{})
	for _, typ := range types {
		lookup[xxhash.Checksum64([]byte(typ))] = struct{}{}
	}

	// Iterate over all keys within the events index which are prefixed with the right block height.
	var events []flow.Event
	prefix := make([]byte, 1+8)
	prefix[0] = prefixDataEvents
	binary.BigEndian.PutUint64(prefix[1:1+8], height)
	err := c.core.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		// NOTE: this is an optimization only, it does not enforce that all
		// results in the iteration have this prefix.
		opts.Prefix = prefix

		it := tx.NewIterator(opts)
		defer it.Close()

		// Iterate on all keys with the right prefix.
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			// If types were given for filtering, discard events which should not be included.
			hash := binary.BigEndian.Uint64(it.Item().Key()[1+8:])
			_, ok := lookup[hash]
			if len(lookup) != 0 && !ok {
				continue
			}

			// Unmarshal event batch and append them to result slice.
			var evts []flow.Event
			err := it.Item().Value(func(val []byte) error {
				val, err := c.core.decompressor.DecodeAll(val, nil)
				if err != nil {
					return fmt.Errorf("could not decompress events: %w", err)
				}
				err = cbor.Unmarshal(val, &evts)
				if err != nil {
					return fmt.Errorf("could not decode events: %w", err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("could not unmarshal events: %w", err)
			}

			events = append(events, evts...)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	return events, nil
}
