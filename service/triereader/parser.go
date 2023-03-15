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

package triereader

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
)

// WalParser is a component that retrieves trie updates and feeds them to its consumer.
type WalParser struct {
	reader WALReader
}

// FromWAL creates a trie update triereader that sources state deltas from a WAL reader.
func FromWAL(reader WALReader) *WalParser {

	f := WalParser{
		reader: reader,
	}

	return &f
}

// AllUpdates returns all updates for the entire spork in wal.
func (f *WalParser) AllUpdates() ([]*ledger.TrieUpdate, error) {

	// We read in a loop because the WAL contains entries that are not trie
	// updates; we don't really need to care about them, so we can just skip
	// them until we find a trie update.
	updates := make([]*ledger.TrieUpdate, 0)
	for f.reader.Next() {
		record := f.reader.Record()
		operation, _, update, err := wal.Decode(record)
		if err != nil {
			return nil, fmt.Errorf("could not decode record: %w", err)
		}
		if operation != wal.WALUpdate {
			continue
		}

		// For older versions, we need to verify the length of types that are aliased
		// to the hash.Hash type from Flow Go, because it is a slice instead of
		// a fixed-length byte array.
		if len(update.RootHash) != 32 {
			return nil, fmt.Errorf("invalid ledger root hash length in trie update: got %d want 32", len(update.RootHash))
		}
		for _, path := range update.Paths {
			if len(path) != 32 {
				return nil, fmt.Errorf("invalid ledger path length in trie update: got %d want 32", len(path))
			}
		}

		// However, we need to make sure that all slices are copied, because the
		// decode function will reuse the underlying slices later.
		update = clone(update)
		updates = append(updates, update)
	}
	err := f.reader.Err()
	if err != nil {
		return nil, fmt.Errorf("could not read next record: %w", err)
	}
	return updates, nil
}
