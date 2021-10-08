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

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/klauspost/compress/zstd"
)

// benchmarkDictionary selects a dataset of samples to compress using the given dictionary,
// and calculates its compression rate and the time it took to compress the given samples.
// It then sets that information directly into the given dictionary pointer.
func (g *Generator) benchmarkDictionary(dict *dictionary) error {

	samples, err := g.getSamples(dict.kind, 10000)
	if err != nil {
		return fmt.Errorf("could not retrieve samples: %w", err)
	}

	// When given an empty dictionary, we're testing the baseline compressing, so we don't want to
	// use a dictionary. Otherwise, use the given dictionary.
	var compressor *zstd.Encoder
	if len(dict.raw) == 0 {
		compressor, err = zstd.NewWriter(nil)
		if err != nil {
			return fmt.Errorf("could not create baseline zstd writer: %w", err)
		}
	} else {
		compressor, err = zstd.NewWriter(nil, zstd.WithEncoderDict(dict.raw))
		if err != nil {
			return fmt.Errorf("could not create zstd writer with dictionary: %w", err)
		}
	}

	start := time.Now()

	var compressed, uncompressed int
	for i := 0; i < 50000; i++ {
		// Pick a random sample.
		sample := samples[rand.Int()%len(samples)]

		uncompressed += len(sample)
		compressed += len(compressor.EncodeAll(sample, nil))
	}

	dict.ratio = float64(compressed) / float64(uncompressed)
	dict.duration = time.Since(start)

	g.log.Debug().
		Int("uncompressed_total", uncompressed).
		Int("compressed_total", compressed).
		Float64("compression_ratio", dict.ratio).
		Dur("compression_duration", dict.duration).
		Msg("benchmark successful")

	return nil
}
