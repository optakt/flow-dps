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

package storage

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/OneOfOne/xxhash"
	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

var (
	codec        cbor.EncMode
	compressor   *zstd.Encoder
	decompressor *zstd.Decoder
)

func init() {
	dict, err := hex.DecodeString(dps.Dictionary)
	if err != nil {
		panic(fmt.Errorf("could not decode dictionary"))
	}

	compressor, err = zstd.NewWriter(nil,
		zstd.WithEncoderDict(dict),
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		panic(fmt.Errorf("could not initialize compressor: %w", err))
	}

	decompressor, err = zstd.NewReader(nil,
		zstd.WithDecoderDicts(dict),
	)
	if err != nil {
		panic(fmt.Errorf("could not initialize decompressor: %w", err))
	}

	codec, err = cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		panic(fmt.Errorf("could not initialize codec: %w", err))
	}
}

func RetrieveLastCommit(commit *flow.StateCommitment) func(*badger.Txn) error {
	return retrieve(encodeKey(prefixLastCommit), commit)
}

func RetrieveHeightByCommit(commit flow.StateCommitment, height *uint64) func(*badger.Txn) error {
	return retrieve(encodeKey(prefixIndexCommitToHeight, commit), height)
}

func RetrieveHeightByBlock(blockID flow.Identifier, height *uint64) func(*badger.Txn) error {
	return retrieve(encodeKey(prefixIndexBlockToHeight, blockID), height)
}

func RetrieveCommitByHeight(height uint64, commit *flow.StateCommitment) func(*badger.Txn) error {
	return retrieve(encodeKey(prefixIndexHeightToCommit, height), commit)
}

func RetrieveHeader(height uint64, header *flow.Header) func(*badger.Txn) error {
	return retrieve(encodeKey(prefixDataHeader, height), header)
}

func RetrieveEvents(height uint64, types []flow.EventType, events *[]flow.Event) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		lookup := make(map[uint64]struct{})
		for _, typ := range types {
			hash := xxhash.ChecksumString64(string(typ))
			lookup[hash] = struct{}{}
		}

		prefix := encodeKey(prefixDataEvents, height)
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
				val, err := decompressor.DecodeAll(val, nil)
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

			*events = append(*events, evts...)
		}

		return nil
	}
}

func RetrievePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		key := encodeKey(prefixDataDelta, path, height)
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
			val, err := decompressor.DecodeAll(val, nil)
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
	}
}

func SaveLastCommit(commit flow.StateCommitment) func(*badger.Txn) error {
	return save(encodeKey(prefixLastCommit), commit)
}

func SaveCommitForHeight(commit flow.StateCommitment, height uint64) func(*badger.Txn) error {
	return save(encodeKey(prefixIndexCommitToHeight, commit), height)
}

func SaveHeightForCommit(height uint64, commit flow.StateCommitment) func(*badger.Txn) error {
	return save(encodeKey(prefixIndexHeightToCommit, height), commit)
}

func SaveHeightForBlock(blockID flow.Identifier, height uint64) func(*badger.Txn) error {
	return save(encodeKey(prefixIndexBlockToHeight, blockID), height)
}

func SaveHeaderForHeight(height uint64, header *flow.Header) func(*badger.Txn) error {
	return save(encodeKey(prefixDataHeader, height), header)
}

func SavePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(*badger.Txn) error {
	return save(encodeKey(prefixDataDelta, path, height), payload)
}

func SaveEvents(height uint64, typ flow.EventType, events []flow.Event) func(*badger.Txn) error {
	hash := xxhash.ChecksumString64(string(typ))
	return save(encodeKey(prefixDataEvents, height, hash), events)
}

func retrieve(key []byte, value interface{}) func(tx *badger.Txn) error {
	return func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return fmt.Errorf("unable to retrieve value: %w", err)
		}

		err = item.Value(func(val []byte) error {
			val, err := decompressor.DecodeAll(val, nil)
			if err != nil {
				return fmt.Errorf("unable to decompress value: %w", err)
			}
			err = cbor.Unmarshal(val, value)
			if err != nil {
				return fmt.Errorf("unable to decode value: %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to retrieve value: %w", err)
		}

		return nil
	}
}

func save(key []byte, value interface{}) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		val, err := codec.Marshal(value)
		if err != nil {
			return fmt.Errorf("unable to encode value: %w", err)
		}

		val = compressor.EncodeAll(val, nil)

		return tx.Set(key, val)
	}
}
