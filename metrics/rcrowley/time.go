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

type Time struct {
	sync.Mutex
	title  string
	timers map[string]metrics.Timer
}

func NewTime(log zerolog.Logger, title string, interval time.Duration) *Time {

	t := Time{
		title:  title,
		timers: make(map[string]metrics.Timer),
	}

	go t.output(log, interval)

	return &t
}

func (t *Time) Duration(name string) func() {
	t.Lock()
	defer t.Unlock()
	timer, ok := t.timers[name]
	if !ok {
		timer = metrics.NewTimer()
		t.timers[name] = timer
	}
	now := time.Now()
	return func() {
		timer.UpdateSince(now)
	}
}

func (t *Time) Output(log zerolog.Logger) {
	t.Lock()
	defer t.Unlock()

	log = log.With().Str("title", t.title).Logger()

	totalDuration := time.Duration(0)
	for _, timer := range t.timers {
		duration := time.Duration(timer.Sum())
		totalDuration += duration
	}

	log.Info().
		Str("duration_total", totalDuration.String()).
		Msg("time metrics for all types")

	for name, timer := range t.timers {
		duration := time.Duration(timer.Sum())
		percentage := float64(duration) / float64(totalDuration)
		log.Info().
			Str("name", name).
			Str("duration_count", duration.String()).
			Float64("duration_percentage", percentage).
			Msg("time metrics for one type")
	}
}

func (t *Time) output(log zerolog.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		t.Output(log)
	}
}
