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

package storage_test

import (
	"errors"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-dps/service/storage"
	"github.com/onflow/flow-dps/testing/helpers"
)

func Test_Fallback(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()
	txn := db.NewTransaction(false)

	// This is a success func that is never expected to be called.
	noCallFn := func(txn *badger.Txn) error {
		t.Log("unexpected function call")
		t.FailNow()
		return nil
	}
	successFn := func(txn *badger.Txn) error {
		return nil
	}
	failFn := func(txn *badger.Txn) error {
		return errors.New("fail")
	}

	t.Run("should error if all ops failed", func(t *testing.T) {
		err := storage.Fallback(
			failFn,
			failFn,
			failFn,
			failFn,
		)(txn)

		// shut up the linter
		_ = multierror.Error{}

		assert.Error(t, err)
		merr, ok := err.(*multierror.Error)
		assert.True(t, ok)
		assert.Len(t, merr.Errors, 4)
	})

	t.Run("should not error if fallback succeeds", func(t *testing.T) {
		err := storage.Fallback(
			failFn,
			successFn,
			noCallFn,
			noCallFn,
		)(txn)

		assert.NoError(t, err)
	})

	t.Run("should not error if any fallback succeeds", func(t *testing.T) {
		err := storage.Fallback(
			failFn,
			failFn,
			failFn,
			failFn,
			failFn,
			successFn,
			noCallFn,
			noCallFn,
			noCallFn,
		)(txn)

		assert.NoError(t, err)
	})

	t.Run("should not call second fallback op if first fallback succeeds", func(t *testing.T) {
		err := storage.Fallback(
			failFn,
			failFn,
			failFn,
			successFn,
			noCallFn,
			noCallFn,
			noCallFn,
		)(txn)

		assert.NoError(t, err)
	})

	t.Run("should not call fallback if first op succeeds", func(t *testing.T) {
		err := storage.Fallback(
			successFn,
			noCallFn,
			noCallFn,
			noCallFn,
			noCallFn,
		)(txn)

		assert.NoError(t, err)
	})
}

func TestCombine(t *testing.T) {
	db := helpers.InMemoryDB(t)
	defer db.Close()
	txn := db.NewTransaction(false)

	// This is a success func that is never expected to be called.
	noCallFn := func(txn *badger.Txn) error {
		t.Log("unexpected function call")
		t.FailNow()
		return nil
	}
	successFn := func(txn *badger.Txn) error {
		return nil
	}
	failFn := func(txn *badger.Txn) error {
		return errors.New("fail")
	}

	t.Run("nominal case", func(t *testing.T) {
		calls := 0
		f := func(txn *badger.Txn) error {
			calls++
			return nil
		}
		err := storage.Combine(f, f, f, f, f, f)(txn)

		assert.NoError(t, err)
		assert.Equal(t, calls, 6)
	})

	t.Run("should error if first op fails", func(t *testing.T) {
		err := storage.Combine(
			failFn,
			noCallFn,
			noCallFn,
			noCallFn,
		)(txn)

		assert.Error(t, err)
	})

	t.Run("should error if last op fails", func(t *testing.T) {
		err := storage.Combine(
			successFn,
			failFn,
		)(txn)

		assert.Error(t, err)
	})

	t.Run("should error if any op fails", func(t *testing.T) {
		err := storage.Combine(
			successFn,
			successFn,
			successFn,
			successFn,
			failFn,
			noCallFn,
			noCallFn,
		)(txn)

		assert.Error(t, err)
	})
}
