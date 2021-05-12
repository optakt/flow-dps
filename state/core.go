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
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/model/dps"
)

type Core struct {
	index        *badger.DB
	compressor   *zstd.Encoder
	decompressor *zstd.Decoder
	height       uint64
	commit       flow.StateCommitment
}

func NewCore(dir string) (*Core, error) {

	opts := badger.DefaultOptions(dir).WithLogger(nil)
	index, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not open index database: %w", err)
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
	err = index.View(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte{dps.PrefixLastHeight})
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
		err = index.Update(func(tx *badger.Txn) error {
			height = 0
			val := make([]byte, 8)
			binary.BigEndian.PutUint64(val, height)
			err = tx.Set([]byte{dps.PrefixLastHeight}, val)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("could not set last height: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last height: %w", err)
	}

	var commit flow.StateCommitment
	err = index.View(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte{dps.PrefixLastCommit})
		if err != nil {
			return err
		}
		commit, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		err = index.Update(func(tx *badger.Txn) error {
			trie, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
			if err != nil {
				return err
			}
			commit = trie.RootHash()
			err = tx.Set([]byte{dps.PrefixLastCommit}, commit)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("could not set last commit: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last commit: %w", err)
	}

	c := &Core{
		index:        index,
		compressor:   compressor,
		decompressor: decompressor,
		height:       height,
		commit:       commit,
	}

	return c, nil
}

// Index is used to index a new set of state deltas for the given block.
func (c *Core) Index(height uint64, blockID flow.Identifier, commit flow.StateCommitment, deltas []model.Delta, events []flow.Event) error {

	// let's use a single transaction to make indexing of a new block atomic
	tx := c.index.NewTransaction(true)

	// first, map the block ID to the height for easy lookup later
	key := make([]byte, 1+len(blockID))
	key[0] = dps.PrefixBlockIndex
	copy(key[1:], blockID[:])
	val := make([]byte, 8)
	binary.BigEndian.PutUint64(val, height)
	err := tx.Set(key, val)
	if err != nil {
		return fmt.Errorf("could not persist block index (%w)", err)
	}

	// second, map the commit to the height for easy lookup later
	key = make([]byte, 1+len(commit))
	key[0] = dps.PrefixCommitIndex
	copy(key[1:], commit)
	err = tx.Set(key, val)
	if err != nil {
		return fmt.Errorf("could not persist commit index (%w)", err)
	}

	// we then index the payload for every path that has changed in this block
	for _, delta := range deltas {
		for _, change := range delta {
			key = make([]byte, 1+pathfinder.PathByteSize+8)
			key[0] = dps.PrefixDeltaIndex
			copy(key[1:1+pathfinder.PathByteSize], change.Path)
			binary.BigEndian.PutUint64(key[1+pathfinder.PathByteSize:], height)
			val, err := cbor.Marshal(change.Payload, cbor.CanonicalEncOptions())
			if err != nil {
				return fmt.Errorf("could not encode payload (%w)", err)
			}
			val = c.compressor.EncodeAll(val, nil)
			err = tx.Set(key, val)
			if err != nil {
				return fmt.Errorf("could not persist payload (%w)", err)
			}
		}
	}

	// Finally, we index the events for every height and event type combination in this block.
	// The first step is to group events by type, so that there is one single entry in the index per type, with all
	// events of that type at the given height.
	buckets := make(map[uint64][]flow.Event)
	for _, event := range events {
		hash := xxhash.Checksum64([]byte(event.Type))
		buckets[hash] = append(buckets[hash], event)
	}

	for hash, evts := range buckets {
		// Prefix + Block Height + Type Hash
		key = make([]byte, 1+8+8)
		key[0] = model.PrefixEventIndex
		binary.BigEndian.PutUint64(key[1:1+8], height)
		binary.BigEndian.PutUint64(key[1+8:1+8+8], hash)

		val, err := cbor.Marshal(evts, cbor.CanonicalEncOptions())
		if err != nil {
			return fmt.Errorf("could not encode events: %w", err)
		}
		val = c.compressor.EncodeAll(val, nil)
		err = tx.Set(key, val)
		if err != nil {
			return fmt.Errorf("could not persist events: %w", err)
		}
	}

	// index the latest height/commit
	key = []byte{dps.PrefixLastHeight}
	val = make([]byte, 8)
	binary.BigEndian.PutUint64(val, height)
	err = tx.Set(key, val)
	if err != nil {
		return fmt.Errorf("could not persist last height: %w", err)
	}
	key = []byte{dps.PrefixLastCommit}
	err = tx.Set(key, commit)
	if err != nil {
		return fmt.Errorf("could not persist last commit: %w", err)
	}

	// let's not forget to finalize the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("could not commit transaction (%w)", err)
	}

	c.height = height
	c.commit = commit

	return nil
}

// Last returns the last block height and state commitment indexed.
func (c *Core) Last() (uint64, flow.StateCommitment) {
	return c.height, c.commit
}

// Height returns the first height for a given state commitment.
func (c *Core) Height(commit flow.StateCommitment) (uint64, error) {

	// build the key and look up the height for the commit
	key := make([]byte, 1+len(commit))
	key[0] = dps.PrefixCommitIndex
	copy(key[1:], commit)
	var height uint64
	err := c.index.View(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		_ = item.Value(func(val []byte) error {
			height = binary.BigEndian.Uint64(val)
			return nil
		})
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not look up height for commit: %w", err)
	}

	return height, nil
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
	prefix[0] = model.PrefixEventIndex
	binary.BigEndian.PutUint64(prefix[1:1+8], height)
	err := c.index.View(func(tx *badger.Txn) error {
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

// Payload returns the payload of the given path at the given block height.
func (c *Core) Payload(height uint64, path ledger.Path) (*ledger.Payload, error) {

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
	key[0] = dps.PrefixDeltaIndex
	copy(key[1:1+pathfinder.PathByteSize], path)
	binary.BigEndian.PutUint64(key[1+pathfinder.PathByteSize:], height)
	err := c.index.View(func(tx *badger.Txn) error {
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
