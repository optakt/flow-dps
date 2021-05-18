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
	"github.com/prometheus/client_golang/prometheus"
	pwal "github.com/prometheus/tsdb/wal"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
)

type LedgerWAL struct {
	reader *pwal.Reader
	limit  uint
	queue  *deque.Deque
	cache  map[string]*deque.Deque
}

// FromLedgerWAL creates a trie update feeder that sources state deltas
// directly from an execution node's trie directory.
func FromLedgerWAL(dir string) (*LedgerWAL, error) {

	w, err := pwal.NewSize(
		nil,
		prometheus.DefaultRegisterer,
		dir,
		32*1024*1024,
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize WAL: %w", err)
	}
	segments, err := pwal.NewSegmentsReader(w.Dir())
	if err != nil {
		return nil, fmt.Errorf("could not initialize segments reader: %w", err)
	}

	l := LedgerWAL{
		reader: pwal.NewReader(segments),
		limit:  1200, // some tolerance on top of execution node forest capacity
		queue:  deque.New(1200, 1200),
		cache:  make(map[string]*deque.Deque),
	}

	return &l, nil
}

func (l *LedgerWAL) Clear() {
	for uint(l.queue.Len()) > l.limit {
		commit := l.queue.PopFront().(flow.StateCommitment)
		delete(l.cache, string(commit))
	}

}

func (l *LedgerWAL) Delta(commitRequest flow.StateCommitment) (dps.Delta, error) {

	// If we have a cached delta for the commit, it means that we skipped it at
	// an earlier point and it is part of a branch that we didn't follow. In
	// that case, we should just return it, because it means the mapper is
	// rewinding and switching to another branch, because the one it was one
	// didn't continue.
	forks, ok := l.cache[string(commitRequest)]
	if ok && forks.Len() > 0 {
		delta := forks.PopBack().(dps.Delta)
		return delta, nil
	}

	// Otherwise, we read from the on-disk file until we find a delta that can
	// be applied to the requested commit. When we are on a fork that stopped,
	// it's possible the mapper will request a commit that will never show up
	// in the WAL. We thus need some kind of limit at which we give up.
	peeked := uint(0)
	for {

		// Increase the number of traversed deltas first, in case we need to
		// break out of this loop. If we reach the limit, it means there is
		// no delta for the requested commit.
		peeked++
		if peeked > l.limit {
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
		duplicate := make([]byte, len(record))
		copy(duplicate, record)
		operation, _, update, err := wal.Decode(duplicate)
		if err != nil {
			return nil, fmt.Errorf("could not decode record: %w", err)
		}
		if operation != wal.WALUpdate {
			continue
		}

		// Deduplicate the paths and payloads.
		// sort and deduplicate paths (we only consider the last occurrence, and ignore the rest)
		commitUpdate := flow.StateCommitment(update.RootHash)
		paths := make([]ledger.Path, 0)
		lookup := make(map[string]ledger.Payload)
		for i, path := range update.Paths {
			if _, ok := lookup[string(path)]; !ok {
				paths = append(paths, path)
			}
			lookup[string(path)] = *update.Payloads[i]
		}
		sort.Slice(paths, func(i, j int) bool {
			return bytes.Compare(paths[i], paths[j]) < 0
		})
		payloads := make([]ledger.Payload, 0, len(paths))
		for _, path := range paths {
			payloads = append(payloads, lookup[string(path)])
		}

		// At this point, we can convert the trie update into a delta. If it's
		// a match for the commit that is requested, we return it. Otherwise,
		// we store it in the cache in case there was a fork and we need it
		// later.
		delta := make(dps.Delta, 0, len(update.Paths))
		for index, path := range paths {
			payload := payloads[index]
			change := dps.Change{
				Path:    path,
				Payload: payload,
			}
			delta = append(delta, change)
		}
		if !bytes.Equal(commitUpdate, commitRequest) {
			forks, ok := l.cache[string(commitUpdate)]
			if !ok {
				forks = deque.New()
				l.cache[string(commitUpdate)] = forks
			}
			forks.PushBack(delta)
			l.queue.PushBack(commitUpdate)
			continue
		}

		return delta, nil
	}
}
