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

// DefaultConfig has the default values of the config set.
var DefaultConfig = Config{
	RootCheckpoint: "",
	SkipRegisters:  false,
	WaitInterval:   100 * time.Millisecond,
}

// Config contains optional parameters for the mapper.
type Config struct {
	RootCheckpoint string
	SkipRegisters  bool
	WaitInterval   time.Duration
}

// WithRootCheckpoint provides the path to a root checkpoint file that is
// required when bootstrapping a new index.
func WithRootCheckpoint(checkpoint string) func(*Config) {
	return func(cfg *Config) {
		cfg.RootCheckpoint = checkpoint
	}
}

// WithSkipRegisters can enable skipping the indexing of register payloads. It
// is mostly meant to be used for testing and debuging.
func WithSkipRegister(skip bool) func(*Config) {
	return func(cfg *Config) {
		cfg.SkipRegisters = skip
	}
}

// WithWaitInterval sets the wait interval that we will wait before retrying
// to retrieve a trie update when it wasn't available.
func WithWaitInterval(interval time.Duration) func(*Config) {
	return func(cfg *Config) {
		cfg.WaitInterval = interval
	}
}
