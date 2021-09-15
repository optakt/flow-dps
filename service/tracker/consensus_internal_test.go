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

package tracker

import (
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNewConsensus(t *testing.T) {
	header := mocks.GenericHeader

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		log := zerolog.Nop()
		hold := mocks.BaselineRecordHolder(t)

		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.InsertRootHeight(header.Height)))

		cons, err := NewConsensus(log, db, hold)

		require.NoError(t, err)
		assert.Equal(t, hold, cons.hold)
		assert.Equal(t, db, cons.db)
		assert.Equal(t, header.Height, cons.last)
	})

	t.Run("handles missing root height", func(t *testing.T) {
		t.Parallel()

		log := zerolog.Nop()
		hold := mocks.BaselineRecordHolder(t)

		db := helpers.InMemoryDB(t)
		// Root height missing from database.
		//require.NoError(t, db.Update(operation.InsertRootHeight(header.Height)))

		_, err := NewConsensus(log, db, hold)

		assert.Error(t, err)
	})
}

func TestConsensus_OnBlockFinalized(t *testing.T) {
	header := mocks.GenericHeader

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		require.NoError(t, db.Update(operation.InsertHeader(header.ID(), header)))

		cons := BaselineConsensus(t, WithDB(db))

		cons.OnBlockFinalized(header.ID())

		assert.Equal(t, cons.last, header.Height)
	})

	t.Run("handles missing header in DB", func(t *testing.T) {
		t.Parallel()

		db := helpers.InMemoryDB(t)
		defer db.Close()

		// Header missing from database.
		//require.NoError(t, db.Update(operation.InsertHeader(header.ID(), header)))

		cons := BaselineConsensus(t, WithDB(db))

		cons.OnBlockFinalized(header.ID())

		assert.NotEqual(t, cons.last, header.Height)
	})
}

func BaselineConsensus(t *testing.T, opts ...func(*Consensus)) *Consensus {
	t.Helper()

	log := zerolog.Nop()
	hold := mocks.BaselineRecordHolder(t)

	cons := Consensus{
		// DB is omitted on purpose, since if we set it here it will never be closed properly.
		// Tests that need to use the DB should set it using the WithDB function.
		hold: hold,
		log: log,
	}

	for _, opt := range opts {
		opt(&cons)
	}

	return &cons
}

func WithHolder(hold RecordHolder) func(*Consensus) {
	return func(consensus *Consensus) {
		consensus.hold = hold
	}
}

func WithDB(db *badger.DB) func(*Consensus) {
	return func(consensus *Consensus) {
		consensus.db = db
	}
}

func WithLast(height uint64) func(*Consensus) {
	return func(consensus *Consensus) {
		consensus.last = height
	}
}