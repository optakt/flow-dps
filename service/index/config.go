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

package index

import "time"

// DefaultConfig is the default configuration for the DPS index.
var DefaultConfig = Config{
	ConcurrentTransactions: 16,          // same value as used for batches in badger
	FlushInterval:          time.Second, // how often to flush Badger transactions
}

// Config is the configuration of a DPS index.
type Config struct {
	ConcurrentTransactions uint
	FlushInterval          time.Duration
}

// WithConcurrentTransactions specifies the maximum concurrent transactions
// that a DPS index should have.
func WithConcurrentTransactions(concurrent uint) func(*Config) {
	return func(cfg *Config) {
		cfg.ConcurrentTransactions = concurrent
	}
}

// WithFlushInterval sets a custom interval after which we will flush Badger
// transactions if no new operations are added.
func WithFlushInterval(interval time.Duration) func(*Config) {
	return func(cfg *Config) {
		cfg.FlushInterval = interval
	}
}
