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
	"fmt"

	pwal "github.com/prometheus/tsdb/wal"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"

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

func (l *LedgerWAL) Update() (*ledger.TrieUpdate, error) {

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

		return update, nil
	}
}
