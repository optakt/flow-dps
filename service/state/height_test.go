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

	"github.com/onflow/flow-go/model/flow"
)

func TestHeight_ForBlock(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	h := c.Height()

	t.Run("should return matching height for blockID", func(t *testing.T) {
		gotHeight, err := h.ForBlock(testBlockID)
		assert.NoError(t, err)
		assert.Equal(t, lastHeight-1, gotHeight)
	})

	t.Run("should return error for unindexed blockID", func(t *testing.T) {
		_, err := h.ForBlock(flow.Identifier{})
		assert.Error(t, err)
	})
}

func TestHeight_ForCommit(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	h := c.Height()

	t.Run("should return matching height for commit", func(t *testing.T) {
		gotHeight, err := h.ForCommit(lastCommit)
		assert.NoError(t, err)
		assert.Equal(t, lastHeight, gotHeight)
	})

	t.Run("should return error for unindexed commit", func(t *testing.T) {
		_, err := h.ForCommit(flow.StateCommitment{})
		assert.Error(t, err)
	})
}
