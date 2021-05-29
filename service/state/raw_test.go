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
)

func TestRaw_WithHeight(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db}
	r := c.Raw().WithHeight(47)

	raw, ok := r.(*Raw)
	require.True(t, ok)

	assert.Equal(t, uint64(47), raw.height)
}

func TestRaw_Get(t *testing.T) {
	db := inMemoryDB(t)
	defer db.Close()

	c := &Core{db: db, height: lastHeight}
	r := c.Raw().WithHeight(lastHeight)

	t.Run("nominal case", func(t *testing.T) {
		got, err := r.Get(testKeyHex)
		assert.NoError(t, err)
		assert.Equal(t, testValue, got)
	})

	t.Run("invalid key format", func(t *testing.T) {
		got, err := r.Get([]byte(`invalid key`))
		assert.Error(t, err)
		assert.Nil(t, got)
	})
}
