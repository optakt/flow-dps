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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-dps/testing/mocks"
)

func TestEmptyState(t *testing.T) {
	f := mocks.BaselineForest(t, true)
	s := EmptyState(f)

	assert.Equal(t, f, s.forest)
	assert.Equal(t, StatusInitialize, s.status)
	assert.Equal(t, s.height, uint64(math.MaxUint64))
	assert.Zero(t, s.last)
	assert.Zero(t, s.next)
	assert.NotNil(t, s.registers)
	assert.Empty(t, s.registers)
	assert.NotNil(t, s.done)
}
