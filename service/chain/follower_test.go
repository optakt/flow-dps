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

package chain_test

import (
	"io"
	"math"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/chain"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestFollower_Root(t *testing.T) {
	db := populateDB(t)
	defer db.Close()
	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	root, err := c.Root()
	assert.NoError(t, err)
	assert.Equal(t, mocks.GenericHeight, root)
}

func TestFollower_Header(t *testing.T) {
	db := populateDB(t)
	defer db.Close()
	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	header, err := c.Header(mocks.GenericHeight)
	assert.NoError(t, err)

	require.NotNil(t, header)
	assert.Equal(t, dps.FlowMainnet, header.ChainID)

	_, err = c.Header(math.MaxUint64)
	assert.Error(t, err)
}

func TestFollower_Commit(t *testing.T) {
	db := populateDB(t)
	defer db.Close()
	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	commit, err := c.Commit(mocks.GenericHeight)
	assert.NoError(t, err)
	assert.Equal(t, mocks.GenericCommit(0), commit)

	_, err = c.Commit(math.MaxUint64)
	assert.Error(t, err)
}

func TestFollower_Events(t *testing.T) {
	db := populateDB(t)
	defer db.Close()
	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	events, err := c.Events(mocks.GenericHeight)
	assert.NoError(t, err)
	assert.Len(t, events, 2)

	_, err = c.Events(math.MaxUint64)
	assert.Error(t, err)
}

func TestFollower_Collections(t *testing.T) {
	db := populateDB(t)
	defer db.Close()
	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	tt, err := c.Collections(mocks.GenericHeight)
	assert.NoError(t, err)
	assert.Len(t, tt, 2)

	_, err = c.Collections(math.MaxUint64)
	assert.Error(t, err)
}

func TestFollower_Guarantees(t *testing.T) {
	db := populateDB(t)
	defer db.Close()
	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	tt, err := c.Guarantees(mocks.GenericHeight)
	assert.NoError(t, err)
	assert.Len(t, tt, 2)

	_, err = c.Guarantees(math.MaxUint64)
	assert.Error(t, err)
}

func TestFollower_Transactions(t *testing.T) {
	db := populateDB(t)
	defer db.Close()
	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	tt, err := c.Transactions(mocks.GenericHeight)
	assert.NoError(t, err)
	assert.Len(t, tt, 4)

	_, err = c.Transactions(math.MaxUint64)
	assert.Error(t, err)
}

func TestFollower_TransactionResults(t *testing.T) {
	db := populateDB(t)
	defer db.Close()
	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	tr, err := c.Results(mocks.GenericHeight)
	assert.NoError(t, err)
	assert.Len(t, tr, 4)

	_, err = c.Results(math.MaxUint64)
	assert.Error(t, err)
}

func TestFollower_Seals(t *testing.T) {
	db := populateDB(t)
	defer db.Close()

	log := zerolog.New(io.Discard)
	follower := mocks.BaselineExecutionFollower(t)

	c := chain.FromFollower(log, follower, db)

	seals, err := c.Seals(mocks.GenericHeight)
	assert.NoError(t, err)
	assert.Len(t, seals, 4)

	_, err = c.Seals(math.MaxUint64)
	assert.Error(t, err)
}
