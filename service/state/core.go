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

	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
)

// TODO: improve code comments & documentation throughout the refactored
// DPS architecture & components
// => https://github.com/awfm9/flow-dps/issues/40

type Core struct {
	db           *badger.DB
	compressor   *zstd.Encoder
	decompressor *zstd.Decoder
	codec        cbor.EncMode
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

	codec, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, fmt.Errorf("could not initialize codec: %w", err)
	}

	// TODO: think about refactoring this, especially in regards to the empty
	// trie initialization, once we have switched to the new storage API
	// => https://github.com/awfm9/flow-dps/issues/38

	var height uint64
	var commit flow.StateCommitment
	err = db.View(func(tx *badger.Txn) error {

		// first we get the last commit
		if err := RetrieveLastCommit(&commit)(tx); err != nil {
			return fmt.Errorf("could not retrieve last commit: %w", err)
		}

		// then we get the height associated with it
		var heightVal []byte
		if err := Retrieve(Encode(prefixIndexCommit, commit), &heightVal)(tx); err != nil {
			return fmt.Errorf("could not retrieve height from commit: %w", err)
		}
		height = binary.BigEndian.Uint64(heightVal)
		return nil
	})
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, fmt.Errorf("could not retrieve last commit: %w", err)
	}
	if errors.Is(err, badger.ErrKeyNotFound) {

		// create an empty trie root hash as last commit
		trie, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
		if err != nil {
			return nil, fmt.Errorf("could not initialize empty trie: %w", err)
		}
		commit = trie.RootHash()

		err = db.Update(func(tx *badger.Txn) error {

			// store the empty root hash as last commit
			err = tx.Set(Encode(prefixLastCommit), commit)
			if err != nil {
				return fmt.Errorf("could not persist last commit: %w", err)
			}

			// map the last commit to zero height
			err = tx.Set(Encode(prefixIndexCommit, commit), Encode(height))
			if err != nil {
				return fmt.Errorf("could not persist commit index: %w", err)
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("could not bootstrap last commit: %w", err)
		}
	}

	c := Core{
		db:           db,
		compressor:   compressor,
		decompressor: decompressor,
		codec:        codec,
		height:       height,
		commit:       commit,
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

func (c *Core) Commit() dps.Commit {
	return &Commit{core: c}
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
	key[0] = prefixDataDelta
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
