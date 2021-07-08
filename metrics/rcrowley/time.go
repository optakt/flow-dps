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
)

type Time struct {
	sync.Mutex
	title  string
	timers map[string]metrics.Timer
}

func NewTime(title string) *Time {
	t := Time{
		title:  title,
		timers: make(map[string]metrics.Timer),
	}
	return &t
}

func (t *Time) Duration(name string) func() {
	t.Lock()
	timer, ok := t.timers[name]
	if !ok {
		timer = metrics.NewTimer()
		t.timers[name] = timer
	}
	t.Unlock()
	now := time.Now()
	return func() {
		timer.UpdateSince(now)
	}
}
