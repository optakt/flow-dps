package tracker

import (
	"github.com/onflow/flow-go/engine/execution/ingestion/uploader"
	"github.com/onflow/flow-go/model/flow"
)

// RecordStreamer represents something that can stream block data.
type RecordStreamer interface {
	Next() (*uploader.BlockData, error)
}

// RecordHolder represents something that can be used to request
// block data for a specific block identifier.
type RecordHolder interface {
	Record(blockID flow.Identifier) (*uploader.BlockData, error)
}
