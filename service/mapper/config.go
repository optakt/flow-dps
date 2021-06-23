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

package mapper

import (
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
)

// Config contains optional parameters we can set for the mapper.
type Config struct {
	CheckpointFile string
	PostProcessing func(*trie.MTrie)

	indexAll          bool
	indexBlocks       bool
	indexEvents       bool
	indexHeaders      bool
	indexPayloads     bool
	indexRegisters    bool
	indexTransactions bool
}

// WithCheckpointFile will initialize the mapper's internal trie with the trie
// from the provided checkpoint file.
func WithCheckpointFile(file string) func(*Config) {
	return func(cfg *Config) {
		cfg.CheckpointFile = file
	}
}

// WithPostProcessing will provide a callback that allows post-processing of the
// final state trie.
func WithPostProcessing(post func(*trie.MTrie)) func(*Config) {
	return func(cfg *Config) {
		cfg.PostProcessing = post
	}
}

// WithIndexAll sets up the mapper to build all indexes simultaneously.
func WithIndexAll(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.indexAll = b
	}
}

// WithIndexBlocks sets up the mapper to build the block indexes.
func WithIndexBlocks(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.indexBlocks = b
	}
}

// WithIndexEvents sets up the mapper to build the events index.
func WithIndexEvents(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.indexEvents = b
	}
}

// WithIndexHeaders sets up the mapper to build the headers index.
func WithIndexHeaders(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.indexHeaders = b
	}
}

// WithIndexPayloads sets up the mapper to build the payloads index.
func WithIndexPayloads(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.indexPayloads = b
	}
}

// WithIndexRegisters sets up the mapper to build the registers index.
func WithIndexRegisters(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.indexRegisters = b
	}
}

// WithIndexTransactions sets up the mapper to build the transactions index.
func WithIndexTransactions(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.indexTransactions = b
	}
}
