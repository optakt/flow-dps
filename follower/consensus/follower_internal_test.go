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

package consensus

import (
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNew(t *testing.T) {
	log := zerolog.New(io.Discard)
	exec := mocks.BaselineExecutionFollower(t)
	consensus := mocks.BaselineConsensusFollower(t)
	db := helpers.InMemoryDB(t)

	follower := New(log, exec, consensus, db)

	require.NotNil(t, follower)
	assert.Equal(t, follower.execution, exec)
	assert.Equal(t, follower.consensus, consensus)
	assert.Equal(t, follower.log, log)
	assert.Equal(t, follower.db, db)
}
