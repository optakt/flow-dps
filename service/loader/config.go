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
	"github.com/onflow/flow-dps/service/mapper"
)

// DefaultConfig sets the default configuration for the index loader. It is used
// when no options are specified.
var DefaultConfig = Config{
	TrieInitializer: FromScratch(),
	ExcludeHeight:   ExcludeNone(),
}

// Config contains the configuration options for the index loader.
type Config struct {
	TrieInitializer mapper.Loader
	ExcludeHeight   func(uint64) bool
}

// Option is a configuration option for the index loader. It can be passed to
// the index loader's construction function to set optional parameters.
type Option func(*Config)

// WithInitializer injects an initializer for the execution state trie. It will
// be used to initialize the execution state trie that serves as basis for the
// restore. It can be used with the root checkpoint loader in order to load the
// initial root checkpoint from the loader instead of the index.
func WithInitializer(load mapper.Loader) Option {
	return func(cfg *Config) {
		cfg.TrieInitializer = load
	}
}

// WithExclude injects a function to ignore ledger register updates in the index
// database for certain heights when restoring the execution state trie. It can
// be used to exclude the root height during restoration from the index when the
// root checkpoint is loaded from disk directly.
func WithExclude(exclude Exclude) Option {
	return func(cfg *Config) {
		cfg.ExcludeHeight = exclude
	}
}

// Exclude is a function that returns true when a certain height should be
// excluded from the index trie restoration.
type Exclude func(uint64) bool

// ExcludeNone is an exclude function that processes all heights for index trie
// restoration.
func ExcludeNone() Exclude {
	return func(uint64) bool {
		return false
	}
}

// ExcludeAtOrBelow is an exclude function that ignores heights at or below the
// given threshold height. It can be used with the root height of a protocol
// state to avoid processing the root checkpoint registers during restore.
func ExcludeAtOrBelow(threshold uint64) Exclude {
	return func(height uint64) bool {
		return height <= threshold
	}
}
