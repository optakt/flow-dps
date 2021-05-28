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

func TestCommit_ForHeight(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	commit := c.Commit()

	t.Run("should return matching commit for height", func(t *testing.T) {
		got, err := commit.ForHeight(lastHeight)
		assert.NoError(t, err)
		assert.Equal(t, lastCommit, got)
	})

	t.Run("should return error for unindexed height", func(t *testing.T) {
		got, err := commit.ForHeight(lastHeight * 2)
		assert.Error(t, err)
		assert.Equal(t, flow.StateCommitment{}, got)
	})
}
