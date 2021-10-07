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
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/storage"
)

// Generator generates optimized Zstandard dictionaries and turns them into Go files
// to be used for compression.
type Generator struct {
	cfg   config
	log   zerolog.Logger
	db    *badger.DB
	codec dps.Codec
}

// New returns a new dictionary generator.
func New(log zerolog.Logger, db *badger.DB, codec dps.Codec, opts ...Option) *Generator {

	cfg := DefaultConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	g := Generator{
		log:   log.With().Str("component", "generator").Logger(),
		cfg:   cfg,
		db:    db,
		codec: codec,
	}

	return &g
}

// Dictionary generates and compiles an optimized dictionary of the given kind.
func (g *Generator) Dictionary(kind DictionaryKind) error {
	logger := g.log.With().Str("kind", string(kind)).Logger()

	// Clean up sample folder, in case there are already old samples still present
	// on the filesystem.
	err := os.RemoveAll(g.cfg.samplePath)
	if err != nil {
		return fmt.Errorf("could not clean up sample folder: %w", err)
	}

	// As long as the increase in compression ratio is considered tolerable, this loop
	// generates increasingly bigger dictionaries, multiplying their size by a factor of
	// two at each iteration. In each loop, dictionaries are generated and benchmarked.
	var current, previous *dictionary
	for size := g.cfg.startSize; g.tolerateImprovement(current, previous); size = size * 2 {
		logger := logger.With().Int("size", size).Logger()

		// Set previous dictionary, except on first iteration.
		if current != nil {
			previous = current
		}

		// Generate samples equal in size to 100 times the desired dictionary size.
		err = g.generateSamples(KindPayloads, size*100)
		if err != nil {
			return fmt.Errorf("could not generate payload samples: %w", err)
		}

		// Train a dictionary using those samples.
		dict, err := g.trainDictionary(KindPayloads, size)
		if err != nil {
			return fmt.Errorf("could not generate raw dictionary: %w", err)
		}

		// Benchmark the dictionary's compression ratio and speed.
		err = g.benchmarkDictionary(dict)
		if err != nil {
			return fmt.Errorf("could not benchmark dictionary: %w", err)
		}

		logger.Info().
			Float64("compression_ratio", dict.ratio).
			Dur("compression_duration", dict.speed).
			Msg("generated payload dictionary")
	}

	// Since the loop stopped, this means that the last generated dictionary was
	// unsatisfactory, and that the last tolerated one was the previous one.
	best := previous

	g.log.Info().
		Int("best_size", best.size).
		Float64("best_ratio", best.ratio).
		Dur("best_duration", best.speed).
		Msg("found most optimized dictionary")

	// Compile the dictionary into a proper Go file.
	err = g.compile(best)
	if err != nil {
		return fmt.Errorf("could not compile dictionary into Go file: %w", err)
	}

	// Remove samples from the filesystem.
	err = os.RemoveAll(g.cfg.samplePath)
	if err != nil {
		return fmt.Errorf("could not clean up sample folder: %w", err)
	}

	return nil
}

// tolerateImprovement returns true if the improvement between current and previous is at least equal to the
// configured ratio improvement tolerance.
func (g *Generator) tolerateImprovement(current, previous *dictionary) bool {
	if current == nil || previous == nil {
		return true
	}

	return current.ratio > previous.ratio*(1+g.cfg.ratioImprovementTolerance)
}

// generateSamples generates the right amount of samples to match the given size.
func (g *Generator) generateSamples(kind DictionaryKind, size int) error {

	// Create a directory in which to store the samples.
	dirPath := filepath.Join(g.cfg.samplePath, string(kind))
	err := os.MkdirAll(dirPath, 0644)
	if err != nil {
		return fmt.Errorf("could not create sample path: %w", err)
	}

	// Retrieve samples from the index database.
	samples, err := g.getSamples(kind, size)
	if err != nil {
		return fmt.Errorf("could not retrieve samples: %w", err)
	}

	// Write each sample in a file.
	for i, sample := range samples {
		filename := filepath.Join(dirPath, fmt.Sprint(i))
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("could not create sample file: %w", err)
		}

		_, err = file.Write(sample)
		if err != nil {
			_ = file.Close()
			return fmt.Errorf("could not write sample file: %w", err)
		}

		_ = file.Close()
	}

	return nil
}

// getSamples retrieves the requested total size of samples from the index database.
func (g *Generator) getSamples(kind DictionaryKind, size int) ([][]byte, error) {

	// Create an iterator prefix based on the kind of sample we want.
	// FIXME: Instead of using EncodeKey, do it manually.
	// FIXME: Do we want to keep the prefixes internal in the storage package and to manually write the matching numbers
	//        in this package? This might mean that we'd break this tool if we were to update/reorder the prefixes.
	var prefix []byte
	switch kind {
	case KindPayloads:
		prefix = storage.EncodeKey(storage.PrefixPayload)
	case KindTransactions:
		prefix = storage.EncodeKey(storage.PrefixTransaction)
	case KindEvents:
		// FIXME: Select an event type in the prefix.
		prefix = storage.EncodeKey(storage.PrefixEvents)
	}

	// Go through the entries of the index database until enough samples have been collected.
	samples := make([][]byte, 0, size)
	err := g.db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.IteratorOptions{
			Prefix: prefix,
		})
		defer it.Close()

		it.Seek(prefix)

		var totalBytes int
		for totalBytes <= size {
			key := it.Item().Key()

			// If we're out of entries to read from, reset the iterator.
			// This will result in duplicate entries in the samples, but should not be a big deal.
			if key[0] != storage.PrefixPayload {
				g.log.Info().Msg("reached end of entries in index database, rewinding")

				it.Rewind()
				it.Next()
			}

			// Retrieve the value of the sample.
			val, err := tx.Get(key)
			if err != nil {
				return fmt.Errorf("could not get value from key %x: %w", key, err)
			}

			err = val.Value(func(val []byte) error {
				value, err := g.codec.Decompress(val)
				if err != nil {
					return fmt.Errorf("could not decompress value from key %x: %w", key, err)
				}

				// If for some reason, an empty value is stored at that key,
				// no need to add it to the samples.
				if len(value) == 0 {
					return nil
				}

				samples = append(samples, value)
				totalBytes += len(value)

				return nil
			})
			if err != nil {
				return fmt.Errorf("could not read value from key %x: %w", key, err)
			}

			it.Next()
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not read from index database: %w", err)
	}

	return samples, nil
}
