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

package generator

// DefaultConfig is the default configuration for the Mapper.
var DefaultConfig = config{
	startSize:                 512, // 512kB
	ratioImprovementTolerance: 0.9,
	samplePath:                "./samples",
	dictionaryPath:            "./codec/zbor/",
}

type config struct {
	// The dictionary size in kB to start with when generating dictionaries.
	// Gets multiplied by 2 at each loop.
	startSize int

	// The tolerance for the improvement of compression ratio between each loop. Should be between 0 and 1.
	// For example, a value of 0.1 means that as long as a dictionary is at least 10% more performant than the
	// previously generated one, its size increase is tolerated and the generation loop continues. Only when a
	// dictionary is generated which is not at least 10% more performant than the previous one does the loop
	// stop, and the previous dictionary is selected as the most optimized one.
	ratioImprovementTolerance float64

	// The path in which to store samples that are generated temporarily to be used for training dictionaries.
	samplePath string
	// The path in which to store compiled Go dictionaries. Should point to the package in which they should be used.
	dictionaryPath string
}

// Option is an option that can be given to the generator to configure optional
// parameters on initialization.
type Option func(*config)

// WithStartSize sets the dictionary size in kB to start with when generating dictionaries.
// This value cannot be below 512kB or it will trigger errors in the Zstandard training algorithm.
// See https://github.com/facebook/zstd/issues/2815
func WithStartSize(size int) Option {
	return func(cfg *config) {
		cfg.startSize = size
	}
}

// WithRatioImprovementTolerance sets the total size in bytes of samples to use for benchmarking. Using high values will
// result in a more accurate calculation of the compression ratio at the expense of making benchmarks longer.
func WithRatioImprovementTolerance(tolerance float64) Option {
	return func(cfg *config) {
		cfg.ratioImprovementTolerance = tolerance
	}
}

// WithSamplePath sets path in which to temporarily store generated data samples.
func WithSamplePath(path string) Option {
	return func(cfg *config) {
		cfg.samplePath = path
	}
}

// WithDictionaryPath sets path in which to store compiled dictionaries.
func WithDictionaryPath(path string) Option {
	return func(cfg *config) {
		cfg.dictionaryPath = path
	}
}
