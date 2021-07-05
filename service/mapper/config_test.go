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

	"github.com/stretchr/testify/assert"
)

func TestWithIndexCommit(t *testing.T) {
	c := &Config{}

	WithIndexCommit(true)(c)

	assert.Equal(t, true, c.IndexCommit)
}

func TestWithIndexHeader(t *testing.T) {
	c := &Config{}

	WithIndexHeader(true)(c)

	assert.Equal(t, true, c.IndexHeader)
}

func TestWithIndexTransactions(t *testing.T) {
	c := &Config{}

	WithIndexTransactions(true)(c)

	assert.Equal(t, true, c.IndexTransactions)
}

func TestWithIndexEvents(t *testing.T) {
	c := &Config{}

	WithIndexEvents(true)(c)

	assert.Equal(t, true, c.IndexEvents)
}

func TestWithIndexPayloads(t *testing.T) {
	c := &Config{}

	WithIndexPayloads(true)(c)

	assert.Equal(t, true, c.IndexPayloads)
}
