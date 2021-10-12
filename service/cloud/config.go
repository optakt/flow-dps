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

package cloud

import (
	"github.com/onflow/flow-go/model/flow"
)

// DefaultConfig is the default configuration for the Google Cloud Streamer.
var DefaultConfig = Config{
	BufferSize:    32,
	CatchupBlocks: []flow.Identifier{},
}

// Config is the configuration for a Google Cloud Streamer.
type Config struct {
	BufferSize    uint
	CatchupBlocks []flow.Identifier
}

// Option is a function that can be applied to a Config.
type Option func(*Config)

// WithBufferSize can be used to specify the buffer size for a
// Google Cloud Streamer to use.
func WithBufferSize(size uint) Option {
	return func(cfg *Config) {
		cfg.BufferSize = size
	}
}

// WithCatchupBlocks injects a number of block IDs that are already finalized,
// but for which we still need to download the execution data records.
func WithCatchupBlocks(blockIDs []flow.Identifier) Option {
	return func(cfg *Config) {
		cfg.CatchupBlocks = blockIDs
	}
}
