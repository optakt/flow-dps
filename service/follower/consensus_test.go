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
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/storage/badger/operation"

	"github.com/optakt/flow-dps/service/follower"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestFollower_OnBlockFinalized(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := mocks.GenericIdentifier(0)
		buffer := &bytes.Buffer{}

		log := zerolog.New(buffer)

		db := helpers.InMemoryDB(t)
		require.NoError(t, db.Update(operation.InsertFinalizedHeight(mocks.GenericHeight)))

		// FIXME: inject execution
		follow := follower.NewConsensus(log, db, nil)
		require.NotNil(t, follow)

		follow.OnBlockFinalized(want)

		assert.Empty(t, buffer.Bytes())
	})

	t.Run("no finalized height in db", func(t *testing.T) {
		t.Parallel()

		blockID := mocks.GenericIdentifier(0)
		buffer := &bytes.Buffer{}

		log := zerolog.New(buffer)
		// Do not insert finalized height to trigger the failure.
		db := helpers.InMemoryDB(t)

		// FIXME: inject execution
		follow := follower.NewConsensus(log, db, nil)
		require.NotNil(t, follow)

		follow.OnBlockFinalized(blockID)

		assert.NotEmpty(t, buffer.Bytes())
	})
}
