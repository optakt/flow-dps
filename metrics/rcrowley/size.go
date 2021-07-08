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

	"github.com/rcrowley/go-metrics"
)

type Size struct {
	sync.Mutex
	title   string
	total   map[string]metrics.Counter
	reduced map[string]metrics.Counter
}

func NewSize(title string) *Size {
	s := Size{
		title:   title,
		total:   make(map[string]metrics.Counter),
		reduced: make(map[string]metrics.Counter),
	}
	return &s
}

func (s *Size) Bytes(name string, original int, compressed int) {
	s.Lock()
	total, ok := s.total[name]
	if !ok {
		total = metrics.NewCounter()
		s.total[name] = total
	}
	reduced, ok := s.reduced[name]
	if !ok {
		reduced = metrics.NewCounter()
		s.reduced[name] = reduced
	}
	s.Unlock()
	total.Inc(int64(original))
	reduced.Inc(int64(compressed))
}
