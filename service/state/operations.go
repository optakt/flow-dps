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
	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"
)

// Retrieve gets any arbitrary value from a given key.
func Retrieve(key []byte, value *[]byte) func(txn *badger.Txn) error {
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

// RetrieveCompressed gets any arbitrary compressed value from a given key.
func RetrieveCompressed(decoder *zstd.Decoder, key []byte, value interface{}) func(txn *badger.Txn) error {
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

func SetCompressed(codec cbor.EncMode, encoder *zstd.Encoder, key []byte, value interface{}) func(txn *badger.Txn) error {
	// FIXME: Wouldn't it be better to attach this to the core and make it private, in order to remove the dependency to codec/compressor/decompressor?
	return func(txn *badger.Txn) error {
		val, err := codec.Marshal(value)
		if err != nil {
			return fmt.Errorf("unable to encode value: %w", err)
		}

		val = encoder.EncodeAll(val, nil)

		return txn.Set(key, val)
	}
}
