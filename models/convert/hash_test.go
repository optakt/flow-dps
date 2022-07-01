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

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-dps/models/convert"
	"github.com/onflow/flow-dps/testing/mocks"
)

func TestIDToHash(t *testing.T) {
	blockID := mocks.GenericHeader.ID()
	got := convert.IDToHash(blockID)
	assert.Equal(t, blockID[:], got)
}

func TestCommitToHash(t *testing.T) {
	got := convert.CommitToHash(mocks.GenericCommit(0))
	assert.Equal(t, mocks.ByteSlice(mocks.GenericCommit(0)), got)
}
