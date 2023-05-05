package mocks

import (
	"testing"

	"github.com/onflow/flow-go/engine/execution/ingestion/uploader"
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
