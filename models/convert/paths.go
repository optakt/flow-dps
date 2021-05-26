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
	"github.com/onflow/flow-go/ledger"
)

func PathsToBytes(paths []ledger.Path) [][]byte {
	bb := make([][]byte, 0, len(paths))
	for _, path := range paths {
		b := make([]byte, len(path))
		copy(b, path[:])
		bb = append(bb, b)
	}
	return bb
}

func BytesToPaths(bb [][]byte) ([]ledger.Path, error) {
	paths := make([]ledger.Path, 0, len(bb))
	for _, b := range bb {
		path := ledger.Path(b)
		paths = append(paths, path)
	}
	return paths, nil
}
