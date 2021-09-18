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
	"time"
)

// DefaultConfig is the default configuration for the Mapper.
var DefaultConfig = Config{
	IndexCommit:       false,
	IndexHeader:       false,
	IndexCollections:  false,
	IndexGuarantees:   false,
	IndexTransactions: false,
	IndexResults:      false,
	IndexEvents:       false,
	IndexPayloads:     false,
	IndexSeals:        false,
	SkipBootstrap:     false,
	WaitInterval:      100 * time.Millisecond,
}

// Config contains optional parameters for the Mapper.
type Config struct {
	// The following attributes specify whether to index specific elements.
	IndexCommit       bool
	IndexHeader       bool
	IndexCollections  bool
	IndexGuarantees   bool
	IndexTransactions bool
	IndexResults      bool
	IndexEvents       bool
	IndexPayloads     bool
	IndexSeals        bool

	// Whether to skip the bootstrapping part.
	SkipBootstrap bool

	// The interval of time to wait between each attempt to map blocks when
	// reading from live components.
	WaitInterval time.Duration
}

// WithIndexCommit sets up the mapper to build the commits index.
func WithIndexCommit(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexCommit = do
	}
}

// WithIndexHeader sets up the mapper to build the headers index.
func WithIndexHeader(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexHeader = do
	}
}

// WithIndexCollections sets up the mapper to build the collections index.
func WithIndexCollections(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexCollections = do
	}
}

// WithIndexGuarantees sets up the mapper to build the guarantees index.
func WithIndexGuarantees(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexGuarantees = do
	}
}

// WithIndexTransactions sets up the mapper to build the transactions index.
func WithIndexTransactions(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexTransactions = do
	}
}

// WithIndexResults sets up the mapper to build the transaction results index.
func WithIndexResults(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexResults = do
	}
}

// WithIndexEvents sets up the mapper to build the events index.
func WithIndexEvents(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexEvents = do
	}
}

// WithIndexPayloads sets up the mapper to build the payloads index.
func WithIndexPayloads(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexPayloads = do
	}
}

// WithIndexSeals sets up the mapper to build the seals index.
func WithIndexSeals(do bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexSeals = do
	}
}

// WithSkipBootstrap sets the mapper up to skip indexing the registers from the
// initial checkpoint.
func WithSkipBootstrap(skip bool) func(*Config) {
	return func(cfg *Config) {
		cfg.SkipBootstrap = skip
	}
}

// WithWaitInterval sets the wait interval that we will wait before retrying
// to retrieve a trie update when it wasn't available.
func WithWaitInterval(interval time.Duration) func(*Config) {
	return func(cfg *Config) {
		cfg.WaitInterval = interval
	}
}
