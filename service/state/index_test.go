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

	"github.com/onflow/flow-go/ledger"
)

func TestIndex_Header(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	idx := c.Index()

	err := idx.Header(lastHeight, testHeader)
	assert.NoError(t, err)
}

func TestIndex_Commit(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	idx := c.Index()

	err := idx.Commit(lastHeight, lastCommit)
	assert.NoError(t, err)
}

func TestIndex_Payloads(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	idx := c.Index()

	t.Run("nominal case", func(t *testing.T) {
		err := idx.Payloads(lastHeight, []ledger.Path{testPath}, []*ledger.Payload{testPayload})
		assert.NoError(t, err)
	})

	t.Run("errors when length of paths and payloads mismatch", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("path and payload length mismatch should panic")
			}
		}()

		err := idx.Payloads(lastHeight, []ledger.Path{testPath, testPath}, []*ledger.Payload{testPayload})
		assert.Error(t, err)
	})
}

func TestIndex_Events(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	idx := c.Index()

	err := idx.Events(lastHeight, testEvents)
	assert.NoError(t, err)
}

func TestIndex_Last(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	idx := c.Index()

	err := idx.Last(lastCommit)
	assert.NoError(t, err)
}
