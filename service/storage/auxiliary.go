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
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

// Fallback goes through the provided operations until one of them succeeds.
// If all of them fail, a multi-error with all errors is returned.
func Fallback(ops ...func(*badger.Txn) error) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		var errs error
		for _, op := range ops {
			err := op(tx)
			if err == nil {
				return nil
			}

			errs = multierror.Append(errs, err)
		}

		return errs
	}
}

// Combine goes through the provided operations until one of them fails.
// When the first one fails, the related error is returned.
func Combine(ops ...func(*badger.Txn) error) func(*badger.Txn) error {
	return func(tx *badger.Txn) error {
		for _, op := range ops {
			err := op(tx)
			if err != nil {
				return err
			}
		}

		return nil
	}
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

		compressor := defaultCompressor
		switch value.(type) {
		case *flow.Header:
			compressor = headerCompressor
		case *ledger.Payload:
			compressor = payloadCompressor
		case []flow.Event:
			compressor = eventsCompressor
		}

		val = compressor.EncodeAll(val, nil)

		return tx.Set(key, val)
	}
}
