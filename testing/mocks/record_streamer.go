package mocks

import (
	"testing"

	"github.com/onflow/flow-go/engine/execution/ingestion/uploader"
)

type RecordStreamer struct {
	NextFunc func() (*uploader.BlockData, error)
}

func BaselineRecordStreamer(t *testing.T) *RecordStreamer {
	t.Helper()

	r := RecordStreamer{
		NextFunc: func() (*uploader.BlockData, error) {
			return GenericRecord(), nil
		},
	}

	return &r
}

func (r *RecordStreamer) Next() (*uploader.BlockData, error) {
	return r.NextFunc()
}
