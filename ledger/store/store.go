package store

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v2"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
)

// NOTE: When loading a checkpoint, so many payloads need to be written at once that they, in most cases, get added to
// the store faster than it can persist them on disk. When that is the case and that the LRU cache is full, each call to
// the Save method becomes blocking until the previous entry has been evicted from the cache and persisted on disk,
// which is effectively as slow as always writing on disk for each operation. Under normal operating conditions however,
// this solution allows us to store payloads on disk with a negligible impact to insertion performance and a limited
// impact to memory usage.

// persistInterval is the interval of time at which the store evicts the oldest elements from its LRU cache and stores
// them persistently in the on-disk database.
const persistInterval = 100 * time.Millisecond

// Store is a component that provides fast persistent storage by using an LRU cache from which evicted entries get
// persisted on disk.
type Store struct {
	log zerolog.Logger

	db        *badger.DB
	cache     *lru.Cache
	cacheSize int

	done chan struct{}
}

// NewStore creates a new store using a cache of the given size and storing payloads on disk at the given path.
func NewStore(log zerolog.Logger, size int, storagePath string) (*Store, error) {
	logger := log.With().Str("component", "payload_store").Logger()

	opts := badger.DefaultOptions(storagePath)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not create persistent storage for ledger payloads: %w", err)
	}
	defer db.Close()

	cache, err := lru.NewWithEvict(size, func(key interface{}, value interface{}) {
		hash, ok := key.([]byte)
		if !ok {
			logger.Error().Interface("got", key).Msg("unexpected key format")
		}

		payload, ok := value.([]byte)
		if !ok {
			logger.Error().Interface("got", value).Msg("unexpected value format")
		}

		err = db.Update(func(txn *badger.Txn) error {
			return txn.Set(hash, payload)
		})
	})
	if err != nil {
		return nil, fmt.Errorf("could not create cache storage for ledger payloads: %w", err)
	}

	s := Store{
		log:   logger,
		db:    db,
		cache: cache,
		done:  make(chan struct{}),
	}

	go s.persist()

	return &s, nil
}

// Save stores a payload.
func (s *Store) Save(hash hash.Hash, payload *ledger.Payload) {
	_ = s.cache.Add(hash[:], payload.Value)
}

// Retrieve retrieves a payload from the cache or persistent storage.
func (s *Store) Retrieve(hash hash.Hash) (*ledger.Payload, error) {
	var payload ledger.Payload

	// If the value is still in the cache, fetch it there.
	val, ok := s.cache.Get(hash[:])
	if ok {
		payload.Value = val.([]byte)
		return &payload, nil
	}

	// Otherwise, assume it has been evicted from the cache and persisted on disk.
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(hash[:])
		if err != nil {
			return fmt.Errorf("could not find payload: %w", err)
		}

		_, err = item.ValueCopy(payload.Value)
		if err != nil {
			return fmt.Errorf("could not read payload: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &payload, nil
}

// Persist periodically checks whether the cache is over half full, and if so, persist some of its entries until it is
// no longer half full. If the cache gets full, calls to `Add` become much slower, so it is good to preemptively persist
// part of it regularly.
func (s *Store) persist() {
	ticker := time.NewTicker(persistInterval)
	for {
		select {
		case <-s.done:
			return

		case <-ticker.C:
			// If the cache is less than half full, do nothing.
			if s.cache.Len() < s.cacheSize/2 {
				continue
			}

			// If cache is at least half full, persist its oldest entries until it is only half full.
			for i := 0; i < s.cache.Len()-s.cacheSize/2; i++ {
				key, value, ok := s.cache.GetOldest()
				if !ok {
					s.log.Error().Msg("could not get oldest cached payload")
				}

				hash, ok := key.([]byte)
				if !ok {
					s.log.Error().Interface("got", key).Msg("unexpected key format")
				}

				payload, ok := value.([]byte)
				if !ok {
					s.log.Error().Interface("got", value).Msg("unexpected value format")
				}

				err := s.db.Update(func(txn *badger.Txn) error {
					return txn.Set(hash, payload)
				})
				if err != nil {
					s.log.Error().Msg("could not persist ledger payload")
				}
			}
		}
	}
}

// Stop stops the store's persistence goroutine.
func (s *Store) Stop() {
	close(s.done)
}
