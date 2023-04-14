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
	BootstrapState: false,
	SkipRegisters:  false,
	WaitInterval:   100 * time.Millisecond,
}

// Config contains optional parameters for the Mapper.
type Config struct {
	BootstrapState bool
	SkipRegisters  bool
	WaitInterval   time.Duration
}

// Option is an option that can be given to the mapper to configure optional
// parameters on initialization.
type Option func(*Config)

// WithBootstrapState makes the mapper bootstrap the state from a root
// checkpoint. If not set, it will resume indexing from a previous trie.
func WithBootstrapState(bootstrap bool) Option {
	return func(cfg *Config) {
		cfg.BootstrapState = bootstrap
	}
}

// WithSkipRegisters makes the mapper skip indexing of all ledger registers,
// which speeds up the run significantly and can be used for debugging purposes.
func WithSkipRegisters(skip bool) Option {
	return func(cfg *Config) {
		cfg.SkipRegisters = skip
	}
}

// WithWaitInterval sets the wait interval that we will wait before retrying
// to retrieve a trie update when it wasn't available.
func WithWaitInterval(interval time.Duration) Option {
	return func(cfg *Config) {
		cfg.WaitInterval = interval
	}
}
