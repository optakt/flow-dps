// Copyright 2021 Optakt Labs OÜ
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
	"github.com/hashicorp/go-multierror"
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

func (l *Library) retrieve(key []byte, v interface{}) func(tx *badger.Txn) error {
	// NOTE: When retrieving things from the database, it's important that the
	// variable is initialized within the loop body if the retrieval happens as
	// part of a loop. This makes sure that the value we decode into always has
	// its own independent memory location.
	return func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return fmt.Errorf("could not get value (key: %x): %w", key, err)
		}

		err = item.Value(func(val []byte) error {
			return l.codec.Unmarshal(val, v)
		})
		if err != nil {
			return fmt.Errorf("could not decode value (key: %x): %w", key, err)
		}

		return nil
	}
}

func (l *Library) save(key []byte, value interface{}) func(*badger.Txn) error {
	// NOTE: We want to encode the value right away, rather than doing it inside
	// of the closure. Otherwise, if value is a loop variable, it might not be
	// the same underlying value anymore when iterating through the list by the
	// time that the closure is called in the Badger transaction.
	val, err := l.codec.Marshal(value)
	return func(tx *badger.Txn) error {
		if err != nil {
			return fmt.Errorf("could not encode value (key: %x): %w", key, err)
		}

		err = tx.Set(key, val)
		if err != nil {
			return fmt.Errorf("could not set value (key: %x): %w", key, err)
		}

		return nil
	}
}
