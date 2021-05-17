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
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
)

var (
	codec cbor.EncMode
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

func RetrieveCommitHeight(commit []byte, height *uint64) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		var value []byte
		err := retrieve(Encode(prefixIndexCommit, commit), &value)(txn)
		if err != nil {
			return fmt.Errorf("unable to retrieve commit height: %w", err)
		}

		*height = binary.BigEndian.Uint64(value)

		return nil
	}
}

func RetrieveBlockHeight(blockID []byte, height *uint64) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		var value []byte
		err := retrieve(Encode(prefixIndexHeight, blockID), &value)(txn)
		if err != nil {
			return fmt.Errorf("unable to retrieve block height: %w", err)
		}

		*height = binary.BigEndian.Uint64(value)

		return nil
	}
}

func RetrieveCommit(height uint64, commit *[]byte) func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return retrieveCompressed(Encode(prefixIndexHeight, height), &commit)(txn)
	}
}

func RetrieveHeader(height uint64, header *flow.Header) func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return retrieveCompressed(Encode(prefixDataHeader, height), &header)(txn)
	}
}

func SetLastCommit(commit []byte) func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return txn.Set(Encode(prefixLastCommit), commit)
	}
}

func SetCommitHeight(commit []byte, height uint64) func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return  txn.Set(Encode(prefixIndexCommit, commit), Encode(height))
	}
}

func SetHeightCommit(height uint64, commit []byte) func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return  txn.Set(Encode(prefixIndexHeight, height), commit)
	}
}

func SetBlockHeight(blockID []byte, height uint64) func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return  txn.Set(Encode(prefixIndexBlock, blockID), Encode(height))
	}
}

func SetHeader(height uint64, header *flow.Header) func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return setCompressed(Encode(prefixDataHeader, height), header)(txn)
	}
}

func SetDeltas(height uint64, change dps.Change)func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return setCompressed(Encode(prefixDataDelta, change.Path, height), change.Payload)(txn)
	}
}

func SetEvents(height, typeHash uint64, events []flow.Event) func (txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		return setCompressed(Encode(prefixDataEvents, height, typeHash), events)(txn)
	}
}

func retrieve(key []byte, value *[]byte) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return fmt.Errorf("unable to retrieve value: %w", err)
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("unable to copy value: %w", err)
		}

		*value = val

		return nil
	}
}

func retrieveCompressed(key []byte, value interface{}) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return fmt.Errorf("unable to retrieve value: %w", err)
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("unable to copy value: %w", err)
		}

		val, err = decoder.DecodeAll(val, nil)
		if err != nil {
			return fmt.Errorf("unable to decompress value: %w", err)
		}

		err = cbor.Unmarshal(val, value)
		if err != nil {
			return fmt.Errorf("unable to decode value: %w", err)
		}

		return nil
	}
}

func setCompressed(key []byte, value interface{}) func(txn *badger.Txn) error {
	return func(txn *badger.Txn) error {
		val, err := codec.Marshal(value)
		if err != nil {
			return fmt.Errorf("unable to encode value: %w", err)
		}

		val = encoder.EncodeAll(val, nil)

		return txn.Set(key, val)
	}
}
