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

// DefaultConfig has the default values of the config set.
var DefaultConfig = Config{
	IndexCommit:       false,
	IndexHeader:       false,
	IndexCollections:  false,
	IndexTransactions: false,
	IndexResults:      false,
	IndexEvents:       false,
	IndexPayloads:     false,
}

// Config contains optional parameters for the mapper.
type Config struct {
	IndexCommit       bool
	IndexHeader       bool
	IndexCollections  bool
	IndexTransactions bool
	IndexResults      bool
	IndexEvents       bool
	IndexPayloads     bool
}

// WithIndexCommit sets up the mapper to build the commits index.
func WithIndexCommit(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexCommit = b
	}
}

// WithIndexHeader sets up the mapper to build the headers index.
func WithIndexHeader(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexHeader = b
	}
}

// WithIndexCollections sets up the mapper to build the collections index.
func WithIndexCollections(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexCollections = b
	}
}

// WithIndexTransactions sets up the mapper to build the transactions index.
func WithIndexTransactions(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexTransactions = b
	}
}

// WithIndexResults sets up the mapper to build the transaction results index.
func WithIndexResults(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexResults = b
	}
}

// WithIndexEvents sets up the mapper to build the events index.
func WithIndexEvents(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexEvents = b
	}
}

// WithIndexPayloads sets up the mapper to build the payloads index.
func WithIndexPayloads(b bool) func(*Config) {
	return func(cfg *Config) {
		cfg.IndexPayloads = b
	}
}
