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

package execution_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/mempool/entity"

	"github.com/optakt/flow-dps/follower/execution"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestFollower_OnBlockFinalized(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		log := zerolog.New(io.Discard)
		db := helpers.InMemoryDB(t)
		blocks := mocks.BaselineDownloader(t)

		codec := mocks.BaselineCodec(t)
		blockData := execution.BlockData{
			Block: &flow.Block{
				Header: mocks.GenericHeader,
			},
		}
		codec.UnmarshalFunc = func(_ []byte, value interface{}) error {
			value = &blockData
			return nil
		}

		follower := execution.New(log, blocks, codec, db)

		follower.OnBlockFinalized(mocks.GenericIdentifier(0))

		got := follower.Block()

		require.NotNil(t, got.Block)
		assert.Equal(t, blockData.Block.Header, got.Block.Header)
	})

	t.Run("unmarshal error", func(t *testing.T) {
		t.Parallel()

		buffer := &bytes.Buffer{}
		log := zerolog.New(buffer)
		db := helpers.InMemoryDB(t)
		blocks := mocks.BaselineDownloader(t)

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(_ []byte, value interface{}) error {
			return mocks.GenericError
		}

		follower := execution.New(log, blocks, codec, db)

		follower.OnBlockFinalized(mocks.GenericIdentifier(0))

		got := follower.Block()

		assert.Nil(t, got.Block)
		assert.NotEmpty(t, buffer.Bytes())
	})
}

func TestFollower_IndexAll(t *testing.T) {
	var collections []*entity.CompleteCollection
	for _, guar := range mocks.GenericGuarantees(4) {
		collections = append(collections, &entity.CompleteCollection{
			Guarantee:    guar,
			Transactions: mocks.GenericTransactions(2),
		})
	}

	var events []*flow.Event
	for _, event := range mocks.GenericEvents(4) {
		events = append(events, &event)
	}

	blockData := execution.BlockData{
		Block: &flow.Block{
			Header: mocks.GenericHeader,
			Payload: &flow.Payload{
				Guarantees: mocks.GenericGuarantees(4),
				Seals:      mocks.GenericSeals(4),
			},
		},
		Collections: collections,
		TxResults:   mocks.GenericResults(4),
		Events:      events,
		TrieUpdates: []*ledger.TrieUpdate{mocks.GenericTrieUpdate},
	}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		buffer := &bytes.Buffer{}
		log := zerolog.New(buffer)
		db := helpers.InMemoryDB(t)
		blocks := mocks.BaselineDownloader(t)

		codec := mocks.BaselineCodec(t)
		codec.UnmarshalFunc = func(_ []byte, value interface{}) error {
			value = &blockData
			return nil
		}

		follower := execution.New(log, blocks, codec, db)

		// OnBlockFinalized will populate the follower's block data and subsequently
		// call IndexAll.
		follower.OnBlockFinalized(mocks.GenericIdentifier(0))

		// FIXME: This is a very flaky way to test this func. Maybe this would be better
		//        tested with a proper integration test. Will be done once we are sure
		//        that this design works for us.
		// Assert that no errors occurred which means that the indexing was successful.
		assert.Empty(t, buffer.Bytes())
	})
}
