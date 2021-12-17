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

package store

// Default configuration values.
const (
	DefaultStoragePath = "./payloads"
	DefaultCacheSize   = 4_000_000 // 4MB
)

// Config configures a store.
type Config struct {
	StoragePath string
	CacheSize   int
}

// Option is a function that modifies a configuration.
type Option func(*Config)

// DefaultConfig is the store's default configuration.
var DefaultConfig = Config{
	StoragePath: DefaultStoragePath,
	CacheSize:   DefaultCacheSize,
}

// WithCacheSize specifies the maximum size of the in-memory cache.
func WithCacheSize(size int) Option {
	return func(config *Config) {
		config.CacheSize = size
	}
}

// WithStoragePath specifies the path in which to store ledger payloads on disk.
func WithStoragePath(path string) Option {
	return func(config *Config) {
		config.StoragePath = path
	}
}
