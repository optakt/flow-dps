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

package execution

import (
	"io"
	"math"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/mempool/entity"

	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestFollower_Run(t *testing.T) {
	// Note: This is a minimal test, which I do not want to expand. It is just there to test some of the
	// basic functionality of the execution follower. It is not ideal since it relies on time and might fail
	// if not enough processing power is available.

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		want := fakeBlockData(t)
		fakename := "test"

		codec :=  zbor.NewCodec()
		blockData, err := codec.Marshal(want)
		require.NoError(t, err)

		f, notify := baselineFollower(t)
		blocks := mocks.BaselinePoller(t, notify)
		blocks.ReadFunc = func(name string) ([]byte, error) {
			assert.Equal(t, fakename, name)
			return blockData, nil
		}
		f.blocks = blocks
		f.codec = codec

		go func() {
			err := f.Run()
			assert.NoError(t, err)
		}()

		notify <- fakename

		time.Sleep(50 * time.Millisecond)

		updates, err := f.Update()
		assert.NoError(t, err)
		assert.Equal(t, want.TrieUpdates[0], updates)

		f.Stop()
		select {
		case <-time.After(50 * time.Millisecond):
			t.Fatal("execution follower did not stop within expected time limit")
		case <-f.stop:
			// Follower stopped successfully since it closed its channel.
		}
	})
}

func baselineFollower(t *testing.T) (*Follower, chan string) {
	t.Helper()

	notify := make(chan string)
	blocks := mocks.BaselinePoller(t, notify)
	log := zerolog.New(io.Discard)
	codec := mocks.BaselineCodec(t)
	db := helpers.InMemoryDB(t)

	f := Follower{
		log:    log,
		blocks: blocks,
		codec:  codec,
		db:     db,
		height: math.MaxUint64,
		data:   make(map[uint64]*BlockData, cacheSize),

		stop: make(chan struct{}),
	}

	return &f, notify
}

func fakeBlockData(t *testing.T) BlockData {
	t.Helper()

	var collections []*entity.CompleteCollection
	for _, guar := range mocks.GenericGuarantees(1) {
		collections = append(collections, &entity.CompleteCollection{
			Guarantee:    guar,
			Transactions: mocks.GenericTransactions(2),
		})
	}

	var events []*flow.Event
	for _, event := range mocks.GenericEvents(4) {
		event := event
		events = append(events, &event)
	}

	blockData := BlockData{
		Block: &flow.Block{
			Header: mocks.GenericHeader,
			Payload: &flow.Payload{
				Guarantees: mocks.GenericGuarantees(1),
				Seals:      mocks.GenericSeals(4),
			},
		},
		Collections: collections,
		TxResults:   mocks.GenericResults(4),
		Events:      events,
		TrieUpdates: []*ledger.TrieUpdate{mocks.GenericTrieUpdate},
	}

	return blockData
}
