// Copyright 2021 Alvalor S.A.
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

package feeder

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/gammazero/deque"
	pwal "github.com/prometheus/tsdb/wal"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
)

// TODO: Find a way to clear the cache without discarding needed items.

type LedgerWAL struct {
	reader *pwal.Reader
	count  uint64
	limit  uint
	queue  *deque.Deque
	lookup map[string]*Cache
}

// FromLedgerWAL creates a trie update feeder that sources state deltas
// directly from an execution node's trie directory.
func FromLedgerWAL(dir string) (*LedgerWAL, error) {

	segments, err := pwal.NewSegmentsReader(dir)
	if err != nil {
		return nil, fmt.Errorf("could not initialize segments reader: %w", err)
	}

	l := LedgerWAL{
		reader: pwal.NewReader(segments),
		count:  10,
		limit:  1200, // some tolerance on top of execution node forest capacity
		queue:  deque.New(1200, 1200),
		lookup: make(map[string]*Cache),
	}

	return &l, nil
}

func (l *LedgerWAL) Delta(commitRequest flow.StateCommitment) (dps.Delta, error) {

	// If we have a cached delta for the commit, it means that we skipped it at
	// an earlier point and it is part of an execution branch that we didn't
	// follow yet. We should just return it, because it means the mapper is
	// rewinding and switching to a new execution branch, meaning that the
	// previously returned delta for the same commit was part of a dead branch.
	cache, ok := l.lookup[string(commitRequest)]
	if ok && cache.forks.Len() > 0 {
		delta := cache.forks.PopFront().(dps.Delta)
		return delta, nil
	}

	// Otherwise, we read from the on-disk file until we find a delta that can
	// be applied to the requested commit. When we are on a dead execution fork
	// it's possible the mapper will request a commit that will never show up
	// in the WAL. We thus need some kind of limit at which we give up.
	start := l.count
	for {

		// If we have tried to find the next delta on disk for more than the
		// configured limit, we should give up.
		if l.count >= start+uint64(l.limit) {
			return nil, dps.ErrNotFound
		}

		// Read the next record from the WAL and decode. If it's not a trie
		// update, we skip it, as trie deletes refer to the forest, not
		// registers, and are thus not important for us.
		next := l.reader.Next()
		err := l.reader.Err()
		if !next && err != nil {
			return nil, fmt.Errorf("could not read next record: %w", err)
		}
		if !next {
			return nil, dps.ErrFinished
		}
		record := l.reader.Record()
		operation, _, update, err := wal.Decode(record)
		if err != nil {
			return nil, fmt.Errorf("could not decode record: %w", err)
		}
		if operation != wal.WALUpdate {
			continue
		}

		// Increase the count of deltas read from disk.
		l.count++

		// Deduplicate the paths and payloads. This code is copied from a
		// version of Flow Go where both deduplication and sorting logic where
		// part of the calling logic. In more recent versions, this is done in
		// the called logic, but we want to remain as compatible as possible, at
		// the risk of deduplicating and sorting twice.
		// We also make copies of the paths and payloads, as the code behind the
		// decode function re-uses the underlying slice, which is also re-used
		// by the WAL reader. It also makes it easier to free any space that was
		// allocated around the parts of the overall decoding slice that are
		// actually re-used.
		commitUpdate := flow.StateCommitment(update.RootHash)
		paths := make([]ledger.Path, 0)
		lookup := make(map[string]ledger.Payload)
		for i, path := range update.Paths {
			if _, ok := lookup[string(path)]; !ok {
				paths = append(paths, path.DeepCopy())
			}
			lookup[string(path)] = *update.Payloads[i].DeepCopy()
		}
		sort.Slice(paths, func(i, j int) bool {
			return bytes.Compare(paths[i], paths[j]) < 0
		})
		payloads := make([]ledger.Payload, 0, len(paths))
		for _, path := range paths {
			payloads = append(payloads, lookup[string(path)])
		}

		// At this point, we can convert the trie update into a delta; it is a
		// more efficient DPS format to store on disk.
		delta := make(dps.Delta, 0, len(paths))
		for index, path := range paths {
			payload := payloads[index]
			change := dps.Change{
				Path:    path,
				Payload: payload,
			}
			delta = append(delta, change)
		}

		// If the state commitment of the update doesn't match the requested
		// state commitment, we cache the delta and repeat the loop to read the
		// next one from disk. Cached deltas will potentially be needed for
		// subsequent execution fork resolution.
		if !bytes.Equal(commitUpdate, commitRequest) {
			cache, ok := l.lookup[string(commitUpdate)]
			if !ok {
				cache = &Cache{
					commit: commitUpdate,
					forks:  deque.New(),
					expiry: 0,
				}
				l.queue.PushBack(cache)
				l.lookup[string(commitUpdate)] = cache
			}
			cache.expiry = l.count + uint64(l.limit)
			cache.forks.PushBack(delta)
			continue
		}

		// Clean up all cache items that have expired.
		for l.queue.Len() > 0 {
			cache := l.queue.Front().(*Cache)
			if cache.expiry > l.count {
				break
			}
			_ = l.queue.PopFront()
			delete(l.lookup, string(cache.commit))
		}

		return delta, nil
	}
}
