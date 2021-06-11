// Copyright 2021 Alvalor S.A.
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

	path1, _ := ledger.ToPath(path1b)
	path2, _ := ledger.ToPath(path2b)
	path3, _ := ledger.ToPath(path3b)

	paths := []ledger.Path{path1, path2, path3}

	got := PathsToBytes(paths)

	assert.Equal(t, [][]byte{path1b, path2b, path3b}, got)
}

func TestBytesToPaths(t *testing.T)  {
	t.Run("nominal case", func(t *testing.T) {
		path1b := []byte("aac513eb1a0457700ac3fa8d292513e1")
		path2b := []byte("1454ae2420513f79f6a5e8396d033369")
		path3b := []byte("e91a3eb997752b78bab0bc31e30b1e30")

		path1, _ := ledger.ToPath(path1b)
		path2, _ := ledger.ToPath(path2b)
		path3, _ := ledger.ToPath(path3b)

		wantPaths := []ledger.Path{path1, path2, path3}

		b := [][]byte{path1b, path2b, path3b}

		got, err := BytesToPaths(b)

		assert.NoError(t, err)
		assert.Equal(t, wantPaths, got)
	})

	t.Run("invalid paths should fail", func(t *testing.T) {
		path1b := []byte("aac513eb1a045770") // incorrect length
		path2b := []byte("not even hex") // not hex
		path3b := []byte("") // empty

		b := [][]byte{path1b, path2b, path3b}
		_, err := BytesToPaths(b)

		assert.Error(t, err)
	})
}
