package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/go-multierror"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog"
	"golang.org/x/sync/semaphore"

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

	db    *badger.DB
	sema  *semaphore.Weighted
	tx    *badger.Txn
	mutex *sync.Mutex     // guards the current transaction against concurrent access
	wg    *sync.WaitGroup // keeps track of when the flush goroutine should exit
	err   chan error

	cache     *lru.Cache
	cacheSize int

	done chan struct{}
}

// NewStore creates a new store using a cache of the given size and storing payloads on disk at the given path.
func NewStore(log zerolog.Logger, size int, storagePath string) (*Store, error) {
	logger := log.With().Str("component", "payload_store").Logger()

	opts := badger.DefaultOptions(storagePath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not create persistent storage for ledger payloads: %w", err)
	}

	s := Store{
		log: logger,
		db:  db,
		tx:  db.NewTransaction(true),

		// TODO: Should this be configurable?
		sema:  semaphore.NewWeighted(16),
		err:   make(chan error, 16),
		done:  make(chan struct{}),
		mutex: &sync.Mutex{},
		wg:    &sync.WaitGroup{},
	}

	go s.flush()

	s.cache, err = lru.NewWithEvict(size, func(k interface{}, v interface{}) {
		hash, ok := k.(hash.Hash)
		if !ok {
			logger.Error().Interface("got", k).Msg("unexpected key format")
		}

		value, ok := v.(ledger.Value)
		if !ok {
			logger.Error().Interface("got", v).Msg("unexpected value format")
		}

		err := s.write(hash, value)
		if err != nil {
			logger.Error().Err(err).Msg("could not persist ledger payload")
		}
	})
	if err != nil {
		return nil, fmt.Errorf("could not create cache storage for ledger payloads: %w", err)
	}

	go s.persist()

	return &s, nil
}

// Save stores a payload.
func (s *Store) Save(hash hash.Hash, payload *ledger.Payload) {
	_ = s.cache.Add(hash, payload.Value)
}

// Retrieve retrieves a payload from the cache or persistent storage.
func (s *Store) Retrieve(hash hash.Hash) (*ledger.Payload, error) {
	var payload ledger.Payload

	// If the value is still in the cache, fetch it there.
	val, ok := s.cache.Get(hash)
	if ok {
		payload.Value = val.(ledger.Value)
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

// Close stops the store's persistence goroutines.
func (s *Store) Close() error {

	// Shut down the ticker that makes sure we commit after a certain time
	// without new operations, then drain the tick channel.
	close(s.done)
	s.wg.Wait()

	// The first transaction we created did not claim a slot on the semaphore.
	// This makes sense, because we only want to limit in-flight (committing)
	// transactions. The currently building transaction is not in-progress.
	// However, we still need to make sure that the currently building
	// transaction is properly committed. We assume that we are no longer
	// applying new operations when we call `Close`, so we can explicitly do so
	// here, without using the callback.
	err := s.tx.Commit()
	if err != nil {
		return fmt.Errorf("could not commit final transaction: %w", err)
	}

	// Once we acquire all semaphore resources, it means all transactions have
	// been committed. We can now close the error channel and database and drain
	// any remaining errors.
	_ = s.sema.Acquire(context.Background(), 16)
	s.db.Close()
	close(s.err)

	var merr *multierror.Error
	for err := range s.err {
		merr = multierror.Append(merr, err)
	}

	return merr.ErrorOrNil()
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
				k, v, ok := s.cache.GetOldest()
				if !ok {
					s.log.Error().Msg("could not get oldest cached payload")
				}

				hash, ok := k.(hash.Hash)
				if !ok {
					s.log.Error().Interface("got", k).Msg("unexpected key format")
				}

				value, ok := v.(ledger.Value)
				if !ok {
					s.log.Error().Interface("got", value).Msg("unexpected value format")
				}

				err := s.write(hash, value)
				if err != nil {
					s.log.Error().Err(err).Msg("could not persist ledger payload")
				}
			}
		}
	}
}

// FIXME: This produces an edge case: If a value is evicted from the cache and added to the
//  transaction but not yet committed, it is temporarily no longer available in the store.
//  Therefore, if a new insertion needs to take place and requires the unavailable payload
//  value, it will cause a fatal error.
func (s *Store) write(hash hash.Hash, value ledger.Value) error {
	// Before applying an additional operation to the transaction we are
	// currently building, we want to see if there was an error committing any
	// previous transaction.
	select {
	case err := <-s.err:
		return fmt.Errorf("could not commit transaction: %w", err)
	default:
		// skip
	}

	// Attempt to add a new value in this transaction.
	s.mutex.Lock()
	err := s.tx.Set(hash[:], value[:])
	if errors.Is(err, badger.ErrTxnTooBig) {
		// The transaction is too big already, so it needs to be committed and the operation
		// can be attempted again.
		_ = s.sema.Acquire(context.Background(), 1)
		s.tx.CommitWith(s.committed)
		// Create a new transaction for further operations now that the previous one has been
		// committed.
		s.tx = s.db.NewTransaction(true)
		// Attempt the operation again.
		err = s.tx.Set(hash[:], value[:])
	}
	s.mutex.Unlock()
	if err != nil {
		return fmt.Errorf("could not apply operation: %w", err)
	}


	return nil
}

func (s *Store) committed(err error) {

	// When a transaction is fully committed, we get the result in this
	// callback. In case of an error, we pipe it to the apply function through
	// the error channel.
	if err != nil {
		s.err <- err
	}

	// Releasing one resource on the semaphore frees up one slot for
	// inflight transactions.
	s.sema.Release(1)
}

func (s *Store) flush() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mutex.Lock()
			_ = s.sema.Acquire(context.Background(), 1)
			s.tx.CommitWith(s.committed)
			s.tx = s.db.NewTransaction(true)
			s.mutex.Unlock()

		case <-s.done:
			return
		}
	}
}
