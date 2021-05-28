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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
)

func TestLedger_WithVersion(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	l := c.Ledger().WithVersion(47)

	ldgr, ok := l.(*Ledger)
	require.True(t, ok)

	assert.Equal(t, uint8(47), ldgr.version)
}

func TestLedger_Get(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db, height: lastHeight}
	l := c.Ledger()

	t.Run("nominal case", func(t *testing.T) {
		query, err := ledger.NewQuery(ledger.State(lastCommit), testKeys)
		require.NoError(t, err)

		got, err := l.Get(query)
		assert.NoError(t, err)
		assert.Equal(t, testValues, got)
	})

	t.Run("keys not found", func(t *testing.T) {
		query, err := ledger.NewQuery(ledger.State(lastCommit), []ledger.Key{})
		require.NoError(t, err)

		got, err := l.Get(query)
		assert.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("unknown state", func(t *testing.T) {
		query, err := ledger.NewQuery(ledger.State{}, testKeys)
		require.NoError(t, err)

		got, err := l.Get(query)
		assert.Error(t, err)
		assert.Empty(t, got)
	})
}
