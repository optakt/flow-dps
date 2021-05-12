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
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/OneOfOne/xxhash"
	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor"
	"github.com/klauspost/compress/zstd"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/model/dps"
)

type Core struct {
	db           *badger.DB
	compressor   *zstd.Encoder
	decompressor *zstd.Decoder
	height       uint64
	commit       flow.StateCommitment
}

func NewCore(dir string) (*Core, error) {

	opts := badger.DefaultOptions(dir).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	dict, err := hex.DecodeString(dps.Dictionary)
	if err != nil {
		return nil, fmt.Errorf("could not decode dictionary")
	}

	compressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderDict(dict),
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize compressor: %w", err)
	}

	decompressor, err := zstd.NewReader(nil,
		zstd.WithDecoderDicts(dict),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize decompressor: %w", err)
	}

	var height uint64
	err = db.View(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte{prefixLastHeight})
		if err != nil {
			return err
		}
		_ = item.Value(func(val []byte) error {
			height = binary.BigEndian.Uint64(val)
			return nil
		})
		return nil
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		err = db.Update(func(tx *badger.Txn) error {
			height = 0
			val := make([]byte, 8)
			binary.BigEndian.PutUint64(val, height)
			err = tx.Set([]byte{prefixLastHeight}, val)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("could not set last height: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last height: %w", err)
	}

	c := Core{
		db:           db,
		compressor:   compressor,
		decompressor: decompressor,
		height:       height,
	}

	return &c, nil
}

func (c *Core) Index() dps.Index {
	return &Index{core: c}
}

func (c *Core) Chain() dps.Chain {
	return &Chain{core: c}
}

func (c *Core) Last() dps.Last {
	return &Last{core: c}
}

func (c *Core) Height() dps.Height {
	return &Height{core: c}
}

func (c *Core) Raw() dps.Raw {
	r := Raw{
		core:   c,
		height: c.height,
	}
	return &r
}

func (c *Core) Ledger() dps.Ledger {
	l := Ledger{
		core:    c,
		version: complete.DefaultPathFinderVersion,
	}
	return &l
}

func (c *Core) Events(height uint64, types ...string) ([]flow.Event, error) {
	// Make sure that the request is for a height below the currently active
	// sentinel height; otherwise, we haven't indexed yet and we might return
	// false information.
	if height > c.height {
		return nil, fmt.Errorf("unknown height (current: %d, requested: %d)", c.height, height)
	}

	lookup := make(map[uint64]struct{})
	for _, typ := range types {
		lookup[xxhash.Checksum64([]byte(typ))] = struct{}{}
	}

	// Iterate over all keys within the events index which are prefixed with the right block height.
	var events []flow.Event
	prefix := make([]byte, 1+8)
	prefix[0] = prefixEventData
	binary.BigEndian.PutUint64(prefix[1:1+8], height)
	err := c.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		// NOTE: this is an optimization only, it does not enforce that all
		// results in the iteration have this prefix.
		opts.Prefix = prefix

		it := tx.NewIterator(opts)
		defer it.Close()

		// Iterate on all keys with the right prefix.
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()

			// If types were given for filtering, discard events which should not be included.
			hash := binary.BigEndian.Uint64(key[1+8:])
			_, ok := lookup[hash]
			if len(lookup) != 0 && !ok {
				continue
			}

			// Unmarshal event batch and append them to result slice.
			var evts []flow.Event
			err := it.Item().Value(func(val []byte) error {
				val, err := c.decompressor.DecodeAll(val, nil)
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

func (c *Core) payload(height uint64, path ledger.Path) (*ledger.Payload, error) {

	// Make sure that the request is for a height below the currently active
	// sentinel height; otherwise, we haven't indexed yet and we might return
	// false information because we are missing a delta.
	if height > c.height {
		return nil, fmt.Errorf("unknown height (current: %d, requested: %d)", c.height, height)
	}

	// Use seek on Ledger to seek to the next biggest key lower than the key we
	// seek for; this should represent the last update to the path before the
	// requested height and should thus be the payload we care about.
	var payload ledger.Payload
	key := make([]byte, 1+pathfinder.PathByteSize+8)
<<<<<<< HEAD
	key[0] = prefixDeltaData
=======
	key[0] = PrefixDeltaData
>>>>>>> dddf6c4 (implement height component for state)
	copy(key[1:1+pathfinder.PathByteSize], path)
	binary.BigEndian.PutUint64(key[1+pathfinder.PathByteSize:], height)
	err := c.db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.IteratorOptions{
			PrefetchSize:   0,
			PrefetchValues: false,
			Reverse:        true,
			AllVersions:    false,
			InternalAccess: false,
			Prefix:         key[:1+pathfinder.PathByteSize],
		})
		defer it.Close()
		it.Seek(key)
		if !it.Valid() {
			return dps.ErrNotFound
		}
		err := it.Item().Value(func(val []byte) error {
			val, err := c.decompressor.DecodeAll(val, nil)
			if err != nil {
				return fmt.Errorf("could not decompress payload: %w", err)
			}
			err = cbor.Unmarshal(val, &payload)
			if err != nil {
				return fmt.Errorf("could not decode payload: %w", err)
			}
			return nil
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve payload: %w", err)
	}

	return &payload, nil
}
