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

package follower_test

import (
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/follower"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestSource_Update(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericTrieUpdate

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.UpdateFunc = func() (*ledger.TrieUpdate, error) {
			return want, nil
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Update()

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles execution follower failure", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.UpdateFunc = func() (*ledger.TrieUpdate, error) {
			return nil, mocks.GenericError
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Update()

		assert.Error(t, err)
	})
}

func TestSource_Root(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericHeight

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		cons := mocks.BaselineConsensusFollower(t)

		// Insert root height in DB.
		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.InsertRootHeight(mocks.GenericHeight)))

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Root()

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles missing root height in database", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		cons := mocks.BaselineConsensusFollower(t)

		// Insert nothing in DB.
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Root()

		assert.Error(t, err)
	})
}

func TestSource_Commit(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericCommit(0)

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		cons := mocks.BaselineConsensusFollower(t)
		cons.BlockIDFunc = func() flow.Identifier {
			return mocks.GenericIdentifier(0)
		}

		// Insert state commitment in DB.
		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.IndexStateCommitment(mocks.GenericIdentifier(0), want)))

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Commit(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles unavailable height", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericCommit(0)

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		cons := mocks.BaselineConsensusFollower(t)

		// Insert state commitment in DB.
		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.IndexStateCommitment(mocks.GenericIdentifier(0), want)))

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Commit(mocks.GenericHeight + 999)

		assert.Error(t, err)
	})

	t.Run("handles missing state commitment in database", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		cons := mocks.BaselineConsensusFollower(t)

		// Insert nothing in DB.
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Commit(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestSource_Header(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericHeader

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.HeaderFunc = func(uint64) (*flow.Header, error) {
			return want, nil
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Header(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles execution follower failure", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Header(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestSource_Collections(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericCollections(4)

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.CollectionsFunc = func(uint64) ([]*flow.LightCollection, error) {
			return want, nil
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Collections(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles execution follower failure", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.CollectionsFunc = func(uint64) ([]*flow.LightCollection, error) {
			return nil, mocks.GenericError
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Collections(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestSource_Guarantees(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericGuarantees(4)

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.GuaranteesFunc = func(uint64) ([]*flow.CollectionGuarantee, error) {
			return want, nil
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Guarantees(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles execution follower failure", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.GuaranteesFunc = func(uint64) ([]*flow.CollectionGuarantee, error) {
			return nil, mocks.GenericError
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Guarantees(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestSource_Transactions(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericTransactions(4)

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.TransactionsFunc = func(uint64) ([]*flow.TransactionBody, error) {
			return want, nil
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Transactions(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles execution follower failure", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.TransactionsFunc = func(uint64) ([]*flow.TransactionBody, error) {
			return nil, mocks.GenericError
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Transactions(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestSource_Results(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericResults(4)

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.ResultsFunc = func(uint64) ([]*flow.TransactionResult, error) {
			return want, nil
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Results(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles execution follower failure", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.ResultsFunc = func(uint64) ([]*flow.TransactionResult, error) {
			return nil, mocks.GenericError
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Results(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestSource_Seals(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericSeals(4)

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.SealsFunc = func(uint64) ([]*flow.Seal, error) {
			return want, nil
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Seals(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles execution follower failure", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.SealsFunc = func(uint64) ([]*flow.Seal, error) {
			return nil, mocks.GenericError
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Seals(mocks.GenericHeight)

		assert.Error(t, err)
	})
}

func TestSource_Events(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericEvents(4)

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.EventsFunc = func(uint64) ([]flow.Event, error) {
			return want, nil
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		got, err := s.Events(mocks.GenericHeight)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("handles execution follower failure", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		exec := mocks.BaselineExecutionFollower(t)
		exec.EventsFunc = func(uint64) ([]flow.Event, error) {
			return nil, mocks.GenericError
		}
		cons := mocks.BaselineConsensusFollower(t)
		db := helpers.InMemoryDB(t)

		s := follower.NewSource(log, exec, cons, db)

		_, err := s.Events(mocks.GenericHeight)

		assert.Error(t, err)
	})
}
