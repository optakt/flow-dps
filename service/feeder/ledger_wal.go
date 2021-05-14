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

	"github.com/prometheus/client_golang/prometheus"
	pwal "github.com/prometheus/tsdb/wal"

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
	}

	return &l, nil
}

func (l *LedgerWAL) Delta(commit flow.StateCommitment) (dps.Delta, error) {

	// TODO: fix bug where we have multiple trie updates that should be applied
	// to the same commit (in case of an execution fork)
	// => https://github.com/awfm9/flow-dps/issues/62
	for {
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
		delta := make(dps.Delta, 0, len(update.Paths))
		for index, path := range update.Paths {
			payload := update.Payloads[index]
			change := dps.Change{
				Path:    path,
				Payload: *payload,
			}
			delta = append(delta, change)
		}
		return delta, nil
	}
}
