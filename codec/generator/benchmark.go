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
