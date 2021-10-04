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

package loader

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoader_WithTrieInitializer(t *testing.T) {
	c := Config{
		TrieInitializer: FromCheckpoint(os.Stdin),
	}
	trieInitializer := FromScratch()

	WithInitializer(trieInitializer)(&c)
	assert.Equal(t, trieInitializer, c.TrieInitializer)
}

func TestLoader_WithExclude(t *testing.T) {
	// In Go, the only valid comparison for functions is with nil.
	// Thus we will set ExcludeHeight to nil so we can later verify
	// that it was correctly initialized.
	c := Config{
		ExcludeHeight: nil,
	}
	excluder := ExcludeNone()

	WithExclude(excluder)(&c)
	assert.NotNil(t, c.ExcludeHeight)
}
