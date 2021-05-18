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

	"github.com/awfm9/flow-dps/models/dps"
)

var (
	codec   cbor.EncMode
	encoder *zstd.Encoder
	decoder *zstd.Decoder
)

func init() {
	dict, err := hex.DecodeString(dps.Dictionary)
	if err != nil {
		panic(fmt.Errorf("could not decode dictionary"))
	}

	encoder, err = zstd.NewWriter(nil,
		zstd.WithEncoderDict(dict),
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		panic(fmt.Errorf("could not initialize compressor: %w", err))
	}

	decoder, err = zstd.NewReader(nil,
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

func RetrieveLastCommit(commit *[]byte) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		item, err := txn.Get(Encode(prefixLastCommit))
		if err != nil {
			return fmt.Errorf("unable to retrieve last commit: %w", err)
		}

		_, err = item.ValueCopy(*commit)
		if err != nil {
			return fmt.Errorf("unable to copy last commit: %w", err)
		}

		return nil
	}
}

func RetrieveHeightByCommit(commit []byte, height *uint64) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		var value []byte
		err := retrieve(Encode(prefixIndexCommitToHeight, commit), &value)(txn)
		if err != nil {
			return fmt.Errorf("unable to retrieve commit height: %w", err)
		}

		*height = binary.BigEndian.Uint64(value)

		return nil
	}
}

func RetrieveHeightByBlock(blockID []byte, height *uint64) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		var value []byte
		err := retrieve(Encode(prefixIndexBlockToHeight, blockID), &value)(txn)
		if err != nil {
			return fmt.Errorf("unable to retrieve block height: %w", err)
		}

		*height = binary.BigEndian.Uint64(value)

		return nil
	}
}

func RetrieveCommitByHeight(height uint64, commit *[]byte) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return retrieve(Encode(prefixIndexHeightToCommit, height), &commit)(txn)
	}
}

func RetrieveHeader(height uint64, header *flow.Header) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return retrieve(Encode(prefixDataHeader, height), &header)(txn)
	}
}

func RetrieveEvents(height uint64, types []string, events *[]flow.Event) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		lookup := make(map[uint64]struct{})
		for _, typ := range types {
			lookup[xxhash.Checksum64([]byte(typ))] = struct{}{}
		}

		prefix := Encode(prefixDataEvents, height)
		opts := badger.DefaultIteratorOptions
		// NOTE: this is an optimization only, it does not enforce that all
		// results in the iteration have this prefix.
		opts.Prefix = prefix

		it := txn.NewIterator(opts)
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
				val, err := decoder.DecodeAll(val, nil)
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

func RetrievePayload(height uint64, path ledger.Path, payload *ledger.Payload) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		key := Encode(prefixDataDelta, path, height)
		it := txn.NewIterator(badger.IteratorOptions{
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
			val, err := decoder.DecodeAll(val, nil)
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

func SaveLastCommit(commit []byte) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return save(Encode(prefixLastCommit), commit)(txn)
	}
}

func SaveCommitForHeight(commit []byte, height uint64) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		var heightVal []byte
		binary.BigEndian.PutUint64(heightVal, height)
		return save(Encode(prefixIndexCommitToHeight, commit), heightVal)(txn)
	}
}

func SaveHeightForCommit(height uint64, commit []byte) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return save(Encode(prefixIndexHeightToCommit, height), commit)(txn)
	}
}

func SaveHeightForBlock(blockID []byte, height uint64) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		var heightVal []byte
		binary.BigEndian.PutUint64(heightVal, height)
		return save(Encode(prefixIndexBlockToHeight, blockID), heightVal)(txn)
	}
}

func SaveHeaderForHeight(height uint64, header *flow.Header) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return save(Encode(prefixDataHeader, height), header)(txn)
	}
}

func SaveChangeForHeight(height uint64, change dps.Change) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return save(Encode(prefixDataDelta, change.Path, height), change.Payload)(txn)
	}
}

func SaveEvents(height, typeHash uint64, events []flow.Event) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return save(Encode(prefixDataEvents, height, typeHash), events)(txn)
	}
}

func retrieve(key []byte, value interface{}) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return fmt.Errorf("unable to retrieve value: %w", err)
		}

		err = item.Value(func(val []byte) error {
			val, err := decoder.DecodeAll(val, nil)
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
			return fmt.Errorf("unable to retrieve zalue: %w", err)
		}

		return nil
	}
}

func save(key []byte, value interface{}) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		val, err := codec.Marshal(value)
		if err != nil {
			return fmt.Errorf("unable to encode value: %w", err)
		}

		val = encoder.EncodeAll(val, nil)

		return txn.Set(key, val)
	}
}
