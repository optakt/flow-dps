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

package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"golang.org/x/sync/semaphore"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
	lru "github.com/optakt/golang-lru"
)

// NOTE: When loading a checkpoint, so many payloads need to be written at once that they, in most cases, get added to
// the store faster than it can persist them on disk. When that is the case and that the LRU cache is full, each call to
// the Save method becomes blocking until the previous entry has been evicted from the cache and persisted on disk,
// which is effectively as slow as always writing on disk for each operation. Under normal operating conditions however,
// this solution allows us to store payloads on disk with a negligible impact to insertion performance and a limited
// impact to memory usage.

// Store is a component that provides fast persistent storage by using an LRU cache from which evicted entries get
// persisted on disk.
type Store struct {
	log zerolog.Logger

	db   *badger.DB
	sema *semaphore.Weighted
	tx   *badger.Txn
	txMu *sync.RWMutex   // guards the current transaction against concurrent access
	wg   *sync.WaitGroup // keeps track of when the flush goroutine should exit
	err  chan error

	cache     *lru.Cache
	cacheSize int
	isDirtyMu *sync.RWMutex      // guards the dirty tracking map against concurrent access
	isDirty   map[hash.Hash]bool // keeps track of whether cached entries are on disk

	done chan struct{}
}

// New creates a new store using a cache of the given size and storing payloads on disk at the given path.
func New(log zerolog.Logger, opts ...Option) (*Store, error) {
	logger := log.With().Str("component", "payload_store").Logger()

	config := DefaultConfig
	for _, opt := range opts {
		opt(&config)
	}

	badgerOpts := badger.DefaultOptions(config.StoragePath)
	badgerOpts.Logger = nil
	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, fmt.Errorf("could not create persistent storage for ledger payloads: %w", err)
	}

	s := Store{
		log: logger,
		db:  db,
		tx:  db.NewTransaction(true),

		sema: semaphore.NewWeighted(16),
		err:  make(chan error, 16),
		done: make(chan struct{}),
		txMu: &sync.RWMutex{},
		wg:   &sync.WaitGroup{},

		isDirtyMu: &sync.RWMutex{},
		isDirty:   make(map[hash.Hash]bool, config.CacheSize),
	}

	s.wg.Add(1)
	go s.flush()

	s.cache, err = lru.NewWithEvict(config.CacheSize, func(k, v interface{}, used int) bool {
		hash, ok := k.(hash.Hash)
		if !ok {
			logger.Fatal().Interface("got", k).Msg("unexpected key format")
		}

		// If the entry is dirty and the cache is not full, abort eviction.
		s.isDirtyMu.RLock()
		if s.isDirty[hash] && used < s.cacheSize {
			s.isDirtyMu.RUnlock()
			return false
		}
		s.isDirtyMu.RUnlock()

		// If the cache is full of dirty entries, push the current commit to disk and evict the entry.
		if s.cacheFullAndDirty(used) {
			s.forceCommit()
		}

		// If the current entry is clean, we can evict it. Otherwise, abort the eviction.
		return !s.isDirty[hash]
	})
	if err != nil {
		return nil, fmt.Errorf("could not create cache storage for ledger payloads: %w", err)
	}

	return &s, nil
}

// Save stores a payload.
func (s *Store) Save(hash hash.Hash, payload *ledger.Payload) error {
	// Store in cache.
	_ = s.cache.Add(hash, payload.Value)

	// Set state to dirty.
	s.isDirtyMu.Lock()
	s.isDirty[hash] = true
	s.isDirtyMu.Unlock()

	// Write in transaction.
	err := s.write(hash, payload.Value)
	if err != nil {
		return fmt.Errorf("could not write payload to disk: %w", err)
	}

	return nil
}

// Retrieve retrieves a payload from the cache or persistent storage.
func (s *Store) Retrieve(hash hash.Hash) (*ledger.Payload, error) {
	var payload ledger.Payload

	// If the value is still in the cache, fetch it from there.
	val, ok := s.cache.Get(hash)
	if ok {
		payload.Value = val.(ledger.Value)
		return &payload, nil
	}

	// Otherwise, check if it has been evicted from the cache and is in the current transaction or in the DB.
	s.txMu.RLock()
	defer s.txMu.RUnlock()
	it, err := s.tx.Get(hash[:])
	if err != nil {
		return nil, fmt.Errorf("could not read payload %x: %w", hash[:], err)
	}

	payload.Value, err = it.ValueCopy(nil)
	if err != nil {
		return nil, fmt.Errorf("could not read payload %x: %w", hash[:], err)
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
	s.txMu.Lock()
	err := s.tx.Commit()
	s.txMu.Unlock()
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

func (s *Store) write(hash hash.Hash, value ledger.Value) error {
	// Before applying an additional operation to the transaction we are
	// currently building, we want to see if there was an error committing any
	// previous transaction.
	select {
	case err := <-s.err:
		return fmt.Errorf("could not commit transaction: %w", err)
	case <-s.done:
		return nil
	default:
		// skip
	}

	// Attempt to add a new value in this transaction.
	s.txMu.Lock()
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
	s.txMu.Unlock()
	if err != nil {
		return fmt.Errorf("could not apply operation: %w", err)
	}

	return nil
}

// Check if cache is full of dirty entries, in which case it needs to block new entries
// from being added until the current transaction is written to disk.
func (s *Store) cacheFullAndDirty(used int) bool {
	// Check if there is still space in the cache.
	if used < s.cacheSize {
		return false
	}

	// Check if there is any clean entry in the cache that can be evicted.
	s.isDirtyMu.RLock()
	defer s.isDirtyMu.RUnlock()
	for _, isDirty := range s.isDirty {
		if !isDirty {
			return false
		}
	}

	return true
}

// Since the cache is full of dirty entries, we need to commit the current ones
// to disk immediately so that they can be evicted and let new entries take
// their place.
func (s *Store) forceCommit() {
	s.txMu.Lock()
	defer s.txMu.Unlock()

	fmt.Println("Forcing commit: store is full of dirty entries")

	_ = s.sema.Acquire(context.Background(), 1)
	s.tx.CommitWith(s.committed)
	s.tx = s.db.NewTransaction(true)
}

func (s *Store) committed(err error) {

	// When a transaction is fully committed, we get the result in this
	// callback. In case of an error, we pipe it to the apply function through
	// the error channel.
	if err != nil {
		s.err <- err
	}

	// Once the transaction has been committed, the entries in the cache are no
	// longer dirty.
	s.isDirtyMu.Lock()
	for h := range s.isDirty {
		s.isDirty[h] = false
	}
	s.isDirtyMu.Unlock()

	// Releasing one resource on the semaphore frees up one slot for
	// inflight transactions.
	s.sema.Release(1)
}

// flush is in charge of periodically flushing the cache to disk.
func (s *Store) flush() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.txMu.Lock()
			_ = s.sema.Acquire(context.Background(), 1)
			s.tx.CommitWith(s.committed)
			s.tx = s.db.NewTransaction(true)
			s.txMu.Unlock()

		case <-s.done:
			return
		}
	}
}
