package payload

import (
	"fmt"

	"github.com/cockroachdb/pebble"

	"github.com/onflow/flow-archive/service/storage2"
	"github.com/onflow/flow-go/model/flow"
)

const (
	// Size of the block height encoded in the key.
	heightSuffixLen = 8
)

type payloadStorage struct {
	db *pebble.DB
}

// NewPayloadStorage creates a pebble-backed payload storage.
// The reason we use a separate storage for payloads we need a Comparer with a custom Split function.
//
// It needs to access the last available payload with height less or equal to the requested height.
// This means all point-lookups are range scans.
func NewStorage(dbPath string, cacheSize int64) (*payloadStorage, error) {
	// TODO(rbtz): cache metrics
	cache := pebble.NewCache(cacheSize)
	defer cache.Unref()

	opts := storage2.DefaultPebbleOptions(cache, newMVCCComparer())
	db, err := pebble.Open(dbPath, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	return &payloadStorage{
		db: db,
	}, nil
}

// GetPayload returns the most recent updated payload for the given RegisterID.
// "most recent" means the updates happens most recent up the given height.
//
// For example, if there are 2 values stored for register A at height 6 and 11, then
// GetPayload(13, A) would return the value at height 11.
//
// If no payload is found, an empty byte slice is returned.
func (s *payloadStorage) GetPayload(
	height uint64,
	reg flow.RegisterID,
) ([]byte, error) {
	iter := s.db.NewIter(&pebble.IterOptions{
		UseL6Filters: true,
	})
	defer iter.Close()

	encoded := newLookupKey(height, reg).Bytes()
	ok := iter.SeekPrefixGE(encoded)
	if !ok {
		return []byte{}, nil
	}

	binaryValue, err := iter.ValueAndErr()
	if err != nil {
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	return binaryValue, nil
}

// BatchSetPayload sets the given entries in a batch.
func (s *payloadStorage) BatchSetPayload(
	height uint64,
	entries flow.RegisterEntries,
) error {
	batch := s.db.NewBatch()
	defer batch.Close()

	for _, entry := range entries {
		encoded := newLookupKey(height, entry.Key).Bytes()

		err := batch.Set(encoded, entry.Value, nil)
		if err != nil {
			return fmt.Errorf("failed to set key: %w", err)
		}
	}

	err := batch.Commit(pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

// Close closes the storage.
func (s *payloadStorage) Close() error {
	return s.db.Close()
}
