package storage2

import (
	"fmt"
	"path"

	"go.uber.org/multierr"

	"github.com/cockroachdb/pebble"
	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-archive/service/storage2/payload"
)

var _ archive.Library2 = (*library2Impl)(nil)

type library2Impl struct {
	*payload.Storage
}

func NewLibrary2(dir string, blockCacheSize int64) (archive.Library2, error) {
	// TODO(rbtz): cache metrics
	cache := pebble.NewCache(blockCacheSize)
	defer cache.Unref()

	payloadStor, err := payload.NewStorage(
		path.Join(dir, "payload.db"), cache)
	if err != nil {
		return nil, fmt.Errorf("failed to create payload storage: %w", err)
	}

	return &library2Impl{
		Storage: payloadStor,
	}, nil
}

func (l *library2Impl) Close() (err error) {
	multierr.AppendInto(&err, l.Storage.Close())
	return
}
