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

	"github.com/gammazero/deque"
	"github.com/prometheus/client_golang/prometheus"
	pwal "github.com/prometheus/tsdb/wal"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
)

type LedgerWAL struct {
	log       zerolog.Logger
	reader    *pwal.Reader
	cache     map[string]*deque.Deque
	threshold uint
}

// FromLedgerWAL creates a trie update feeder that sources state deltas
// directly from an execution node's trie directory.
func FromLedgerWAL(log zerolog.Logger, dir string) (*LedgerWAL, error) {

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
		log:       log,
		reader:    pwal.NewReader(segments),
		cache:     make(map[string]*deque.Deque),
		threshold: 100,
	}

	return &l, nil
}

func (l *LedgerWAL) Delta(commit flow.StateCommitment) (dps.Delta, error) {

	// If we have a cached delta for the commit, it means that we skipped it at
	// an earlier point and it is part of a branch that we didn't follow. In
	// that case, we should just return it, because it means the mapper is
	// rewinding and switching to another branch, because the one it was one
	// didn't continue.
	forks, ok := l.cache[string(commit)]
	if ok && forks.Len() > 0 {
		delta := forks.PopBack().(dps.Delta)
		l.log.Debug().Hex("commit", commit).Int("num_changes", len(delta)).Msg("returning non-sequential delta from cache")
		return delta, nil
	}

	// Otherwise, we read from the on-disk file until we find a delta that can
	// be applied to the requested commit. When we are on a fork that stopped,
	// it's possible the mapper will request a commit that will never show up
	// in the WAL. We thus need some kind of threshold at which we give up.
	traversed := uint(0)
	for {

		// Increase the number of traversed deltas first, in case we need to
		// break out of this loop. If we reach the threshold, it means there is
		// no delta for the requested commit.
		traversed++
		if traversed > l.threshold {
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

		// At this point, we can convert the trie update into a delta. If it's
		// a match for the commit that is requested, we return it. Otherwise,
		// we store it in the cache in case there was a fork and we need it
		// later.
		delta := make(dps.Delta, 0, len(update.Paths))
		for index, path := range update.Paths {
			payload := update.Payloads[index]
			change := dps.Change{
				Path:    path,
				Payload: *payload,
			}
			delta = append(delta, change)
		}
		if !bytes.Equal(update.RootHash, commit) {
			forks, ok := l.cache[string(update.RootHash)]
			if !ok {
				forks = deque.New(10, 10)
				l.cache[string(update.RootHash)] = forks
			}
			forks.PushBack(delta)
			l.log.Debug().Hex("commit", update.RootHash).Int("num_changes", len(delta)).Msg("added non-sequential delta to cache")
			continue
		}

		return delta, nil
	}
}
