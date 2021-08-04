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

package feeder

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/optakt/flow-dps/bucket"
)

type Bucket struct {
	downloader *bucket.Downloader
}

// FromDownloader creates a trie update feeder that sources state deltas
// directly from an execution node's trie directory.
func FromDownloader(downloader *bucket.Downloader) (*Bucket, error) {

	l := Bucket{
		downloader: downloader,
	}

	return &l, nil
}

func (d *Bucket) Update() (*ledger.TrieUpdate, error) {

	// We read in a loop because the WAL contains entries that are not trie
	// updates; we don't really need to care about them, so we can just skip
	// them until we find a trie update.
	for {
		segment := d.downloader.Next()
		operation, _, update, err := wal.Decode(segment)
		if err != nil {
			return nil, fmt.Errorf("could not decode segment: %w", err)
		}
		if operation != wal.WALUpdate {
			continue
		}

		// However, we need to make sure that all slices are copied, because the
		// decode function will reuse the underlying slices later.
		update = clone(update)

		return update, nil
	}
}
