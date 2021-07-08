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

package rcrowley

import (
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/rs/zerolog"
)

type Size struct {
	sync.Mutex
	title      string
	original   map[string]metrics.Counter
	compressed map[string]metrics.Counter
}

func NewSize(log zerolog.Logger, title string, interval time.Duration) *Size {
	s := Size{
		title:      title,
		original:   make(map[string]metrics.Counter),
		compressed: make(map[string]metrics.Counter),
	}

	go s.output(log, interval)

	return &s
}

func (s *Size) Bytes(name string, originalCount int, compressedCount int) {
	s.Lock()
	defer s.Unlock()
	original, ok := s.original[name]
	if !ok {
		original = metrics.NewCounter()
		s.original[name] = original
	}
	compressed, ok := s.compressed[name]
	if !ok {
		compressed = metrics.NewCounter()
		s.compressed[name] = compressed
	}
	original.Inc(int64(originalCount))
	compressed.Inc(int64(compressedCount))
}

func (s *Size) Output(log zerolog.Logger) {
	s.Lock()
	defer s.Unlock()

	log = log.With().Str("title", s.title).Logger()

	originalTotal := int64(0)
	compressedTotal := int64(0)
	for _, original := range s.original {
		originalCount := original.Count()
		originalTotal += originalCount
	}
	for _, compressed := range s.compressed {
		compressedCount := compressed.Count()
		compressedTotal += compressedCount
	}

	totalRatio := float64(compressedTotal) / float64(originalTotal)
	log.Info().
		Int64("original_total", originalTotal).
		Int64("compressed_total", compressedTotal).
		Float64("ratio", totalRatio).
		Msg("size metrics for all types")

	for name, original := range s.original {
		compressed := s.compressed[name]
		originalCount := original.Count()
		compressedCount := compressed.Count()
		ratio := float64(compressedCount) / float64(originalCount)
		originalPercentage := float64(originalCount) / float64(originalTotal)
		compressedPercentage := float64(compressedCount) / float64(compressedTotal)
		originalTotal += originalCount
		compressedTotal += compressedCount
		log.Info().
			Str("name", name).
			Int64("original_count", originalCount).
			Int64("compressed_count", compressedCount).
			Float64("original_percentage", originalPercentage).
			Float64("compressed_percentage", compressedPercentage).
			Float64("ratio", ratio).
			Msg("size metrics for one type")
	}
}

func (s *Size) output(log zerolog.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		s.Output(log)
	}
}
