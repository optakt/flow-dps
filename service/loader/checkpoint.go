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

package loader

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
)

// Checkpoint is a loader that loads a trie from a LedgerWAL checkpoint file.
type Checkpoint struct {
	filepath string
	log      *zerolog.Logger
}

// FromCheckpointFile creates a loader which loads the trie from the provided
// reader, which should represent a LedgerWAL checkpoint file.
func FromCheckpointFile(filepath string, log *zerolog.Logger) *Checkpoint {

	c := Checkpoint{
		filepath: filepath,
		log:      log,
	}

	return &c
}

// Trie loads the execution state trie from the LedgerWAL root checkpoint.
func (c *Checkpoint) Trie() (*trie.MTrie, error) {

	trees, err := wal.LoadCheckpoint(c.filepath, c.log)

	if err != nil {
		return nil, fmt.Errorf("could not create Trie from checkpoint: %w", err)
	}

	if len(trees) != 1 {
		return nil, fmt.Errorf("should only have one trie in root checkpoint (tries: %d)", len(trees))
	}

	return trees[0], nil
}
