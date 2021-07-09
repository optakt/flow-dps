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

package output

import (
	"sync"
	"time"

	"github.com/optakt/flow-dps/metrics"
	"github.com/rs/zerolog"
)

type Output struct {
	log        zerolog.Logger
	interval   time.Duration
	collectors []metrics.Collector
	done       chan struct{}
	wg         *sync.WaitGroup
}

func New(log zerolog.Logger, interval time.Duration) *Output {
	o := Output{
		log:        log.With().Str("component", "metrics").Logger(),
		interval:   interval,
		collectors: make([]metrics.Collector, 0, 3),
		done:       make(chan struct{}),
		wg:         &sync.WaitGroup{},
	}
	return &o
}

func (o *Output) Run() {
	o.wg.Add(1)
	go o.loop()
}

func (o *Output) Register(collector metrics.Collector) {
	o.collectors = append(o.collectors, collector)
}

func (o *Output) Stop() {
	close(o.done)
	o.wg.Wait()
}

func (o *Output) loop() {
	defer o.wg.Done()
	ticker := time.NewTicker(o.interval)
Loop:
	for {
		select {
		case <-o.done:
			break Loop
		case <-ticker.C:
			o.print()
		}
	}
	o.print()
	ticker.Stop()
}

func (o *Output) print() {
	for _, collector := range o.collectors {
		collector.Output(o.log)
	}
}
