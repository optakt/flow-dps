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

package mapper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithBootstrapState(t *testing.T) {
	c := Config{
		BootstrapState: false,
	}
	bootstrap := true

	WithBootstrapState(bootstrap)(&c)

	assert.Equal(t, bootstrap, c.BootstrapState)
}

func TestWithSkipRegisters(t *testing.T) {
	c := Config{
		SkipRegisters: false,
	}
	skip := true

	WithSkipRegisters(skip)(&c)

	assert.Equal(t, skip, c.SkipRegisters)
}

func TestWithIndexHeader(t *testing.T) {
	c := &Config{
		WaitInterval: time.Second,
	}
	interval := time.Millisecond

	WithWaitInterval(interval)(c)

	assert.Equal(t, interval, c.WaitInterval)
}
