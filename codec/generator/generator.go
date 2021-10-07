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

func (g *Generator) Dictionary(kind DictionaryKind) error {
	logger := g.log.With().Str("kind", string(kind)).Logger()

	err := os.RemoveAll(g.cfg.samplePath)
	if err != nil {
		return fmt.Errorf("could not clean up sample folder: %w", err)
	}

	var current, previous *dictionary
	for size := g.cfg.startSize; g.tolerateImprovement(current, previous); size = size * 2 {
		logger := logger.With().Int("size", size).Logger()

		// Set previous dictionary, except on first iteration.
		if current != nil {
			previous = current
		}

		err = g.generateSamples(KindPayloads, size*100)
		if err != nil {
			return fmt.Errorf("could not generate payload samples: %w", err)
		}

		dict, err := g.trainDictionary(KindPayloads, size)
		if err != nil {
			return fmt.Errorf("could not generate raw dictionary: %w", err)
		}

		err = g.benchmarkDictionary(dict)
		if err != nil {
			return fmt.Errorf("could not benchmark dictionary: %w", err)
		}

		logger.Info().
			Float64("compression_ratio", dict.ratio).
			Dur("compression_duration", dict.speed).
			Msg("generated payload dictionary")
	}

	best := previous

	g.log.Info().
		Int("best_size", best.size).
		Float64("best_ratio", best.ratio).
		Dur("best_duration", best.speed).
		Msg("found most optimized dictionary")

	err = g.compile(best)
	if err != nil {
		return fmt.Errorf("could not compile dictionary into Go file: %w", err)
	}

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

func (g *Generator) generateSamples(kind DictionaryKind, size int) error {
	dirPath := filepath.Join(g.cfg.samplePath, string(kind))
	err := os.MkdirAll(dirPath, 0644)
	if err != nil {
		return fmt.Errorf("could not create sample path: %w", err)
	}

	samples, err := g.getSamples(kind, size)
	if err != nil {
		return fmt.Errorf("could not retrieve samples: %w", err)
	}

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

func (g *Generator) getSamples(kind DictionaryKind, size int) ([][]byte, error) {
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
			if key[0] != storage.PrefixPayload {
				it.Rewind()
				it.Next()
			}

			val, err := tx.Get(key)
			if err != nil {
				return err
			}

			err = val.Value(func(val []byte) error {
				value, err := g.codec.Decompress(val)
				if err != nil {
					return err
				}

				if len(value) == 0 {
					return nil
				}

				samples = append(samples, value)
				totalBytes += len(value)

				return nil
			})
			if err != nil {
				return err
			}

			it.Next()
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return samples, nil
}
