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

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/ledger"
)

func TestPathsToBytes(t *testing.T) {
	path1b := []byte("aac513eb1a0457700ac3fa8d292513e1")
	path2b := []byte("1454ae2420513f79f6a5e8396d033369")
	path3b := []byte("e91a3eb997752b78bab0bc31e30b1e30")

	var path1, path2, path3 ledger.Path
	copy(path1[:], path1b)
	copy(path2[:], path2b)
	copy(path3[:], path3b)

	paths := []ledger.Path{path1, path2, path3}

	got := PathsToBytes(paths)

	assert.Equal(t, [][]byte{path1b, path2b, path3b}, got)
}

func TestBytesToPaths(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		path1b := []byte("aac513eb1a0457700ac3fa8d292513e1")
		path2b := []byte("1454ae2420513f79f6a5e8396d033369")
		path3b := []byte("e91a3eb997752b78bab0bc31e30b1e30")

		var path1, path2, path3 ledger.Path
		copy(path1[:], path1b)
		copy(path2[:], path2b)
		copy(path3[:], path3b)

		wantPaths := []ledger.Path{path1, path2, path3}

		bb := [][]byte{path1b, path2b, path3b}

		got, err := BytesToPaths(bb)

		assert.NoError(t, err)
		assert.Equal(t, wantPaths, got)
	})

	t.Run("incorrect-length paths should fail", func(t *testing.T) {
		invalidPath := []byte("1a0457700")

		bb := [][]byte{invalidPath}
		_, err := BytesToPaths(bb)

		assert.Error(t, err)
	})

	t.Run("empty paths should fail", func(t *testing.T) {
		invalidPath := []byte("")

		bb := [][]byte{invalidPath}
		_, err := BytesToPaths(bb)

		assert.Error(t, err)
	})

	t.Run("non-hex path should fail", func(t *testing.T) {
		invalidPath := []byte("not even hex")

		bb := [][]byte{invalidPath}
		_, err := BytesToPaths(bb)

		assert.Error(t, err)
	})
}
