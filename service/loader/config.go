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
	"github.com/optakt/flow-dps/service/mapper"
)

var DefaultConfig = Config{
	TrieInitializer: FromScratch(),
	ExcludeHeight:   ExcludeNone(),
}

type Config struct {
	TrieInitializer mapper.Loader
	ExcludeHeight   func(uint64) bool
}

type Option func(*Config)

func WithInitializer(load mapper.Loader) Option {
	return func(cfg *Config) {
		cfg.TrieInitializer = load
	}
}

func WithExclude(exclude Exclude) Option {
	return func(cfg *Config) {
		cfg.ExcludeHeight = exclude
	}
}

type Exclude func(uint64) bool

func ExcludeNone() Exclude {
	return func(uint64) bool {
		return false
	}
}

func ExcludeAtOrBelow(threshold uint64) Exclude {
	return func(height uint64) bool {
		return height <= threshold
	}
}
