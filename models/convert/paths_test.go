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

func TestPathsToBytes(t *testing.T) {
	got := convert.PathsToBytes(mocks.GenericLedgerPaths(3))

	assert.Equal(t, [][]byte{
		mocks.ByteSlice(mocks.GenericLedgerPath(0)),
		mocks.ByteSlice(mocks.GenericLedgerPath(1)),
		mocks.ByteSlice(mocks.GenericLedgerPath(2)),
	}, got)
}

func TestBytesToPaths(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		wantPaths := mocks.GenericLedgerPaths(3)

		bb := [][]byte{
			mocks.ByteSlice(mocks.GenericLedgerPath(0)),
			mocks.ByteSlice(mocks.GenericLedgerPath(1)),
			mocks.ByteSlice(mocks.GenericLedgerPath(2)),
		}

		got, err := convert.BytesToPaths(bb)

		assert.NoError(t, err)
		assert.Equal(t, wantPaths, got)
	})

	t.Run("incorrect-length paths should fail", func(t *testing.T) {
		t.Parallel()

		invalidPath := []byte{0x1a, 0x04, 0x57, 0x70, 0x00}

		bb := [][]byte{invalidPath}
		_, err := convert.BytesToPaths(bb)

		assert.Error(t, err)
	})

	t.Run("empty paths should fail", func(t *testing.T) {
		t.Parallel()

		invalidPath := []byte("")

		bb := [][]byte{invalidPath}
		_, err := convert.BytesToPaths(bb)

		assert.Error(t, err)
	})
}
