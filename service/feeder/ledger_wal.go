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

	pwal "github.com/prometheus/tsdb/wal"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/models/dps"
)

type LedgerWAL struct {
	reader *pwal.Reader
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
	}

	return &l, nil
}

func (l *LedgerWAL) Delta() (*dps.Delta, error) {

	// We read in a loop because the WAL contains entries that are not trie
	// updates; we don't really need to care about them, so we can just skip
	// them until we find a trie update.
	for {

		// This part reads the next entry from the WAL, makes sure we didn't
		// encounter an error when reading or decoding and ensures that it's a
		// trie update.
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

		// Deduplicate the paths and payloads. This code is copied from a
		// version of Flow Go where both deduplication and sorting logic were
		// part of the consumer logic of the trie API. In more recent versions,
		// this is done in the trie logic itself, but we want to remain as
		// compatible as possible, so we might deduplicate and sort twice.
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

		// At this point, we can convert the trie update into a delta; it's
		// slightly different in structure that groups each path with its
		// respective payload, rather than having two different slices.
		delta := dps.Delta{
			Commit:  flow.StateCommitment(update.RootHash),
			Changes: make([]dps.Change, 0, len(paths)),
		}
		for index, path := range paths {
			payload := payloads[index]
			change := dps.Change{
				Path:    path,
				Payload: payload,
			}
			delta.Changes = append(delta.Changes, change)
		}

		return &delta, nil
	}
}
