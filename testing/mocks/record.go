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

package mocks

import (
	"testing"

	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/mempool/entity"
)

type RecordHolder struct {
	RecordFunc func(blockID flow.Identifier) (*uploader.BlockData, error)
}

func BaselineRecordHolder(t *testing.T) *RecordHolder {
	t.Helper()

	r := RecordHolder{
		RecordFunc: func(flow.Identifier) (*uploader.BlockData, error) {
			var collections []*entity.CompleteCollection
			for _, guarantee := range GenericGuarantees(4) {
				collections = append(collections, &entity.CompleteCollection{
					Guarantee:    guarantee,
					Transactions: GenericTransactions(2),
				})
			}

			var events []*flow.Event
			for _, event := range GenericEvents(4) {
				events = append(events, &event)
			}

			data := uploader.BlockData{
				Block: &flow.Block{
					Header: GenericHeader,
					Payload: &flow.Payload{
						Guarantees: GenericGuarantees(4),
						Seals:      GenericSeals(4),
					},
				},
				Collections:          collections,
				TxResults:            GenericResults(4),
				Events:               events,
				TrieUpdates:          GenericTrieUpdates(4),
				FinalStateCommitment: GenericCommit(0),
			}

			return &data, nil
		},
	}

	return &r
}

func (r *RecordHolder) Record(blockID flow.Identifier) (*uploader.BlockData, error) {
	return r.RecordFunc(blockID)
}

type RecordStreamer struct {
	NextFunc func() (*uploader.BlockData, error)
}

func BaselineRecordStreamer(t *testing.T) *RecordStreamer {
	t.Helper()

	r := RecordStreamer{
		NextFunc: func() (*uploader.BlockData, error) {
			return GenericBlockData(), nil
		},
	}

	return &r
}

func (r *RecordStreamer) Next() (*uploader.BlockData, error) {
	return r.NextFunc()
}
