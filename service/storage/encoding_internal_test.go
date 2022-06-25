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

package storage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-dps/testing/mocks"
)

func TestEncodeKey(t *testing.T) {
	id := mocks.GenericHeader.ID()
	path := mocks.GenericLedgerPath(0)
	commit := mocks.GenericCommit(0)
	fullKey := bytes.Join([][]byte{
		{
			0x1,                                     // prefix
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2a, // uint(42)
		},
		id[:],
		path[:],
		commit[:],
	}, nil)

	tests := []struct {
		name string

		segments []interface{}

		wantKey   []byte
		wantPanic bool
	}{
		{
			name: "a key with all types combined should work",

			segments: []interface{}{
				uint64(42),
				id,
				path,
				commit,
			},

			wantPanic: false,
			wantKey:   fullKey,
		},
		{
			name: "empty segments should work",

			wantPanic: false,
			wantKey:   []byte{1},
		},
		{
			name: "unsupported types should panic",

			segments: []interface{}{
				struct{}{},
			},

			wantPanic: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if test.wantPanic {
				assert.Panics(t, func() {
					EncodeKey(1, test.segments...)
				})
				return
			}

			var got []byte
			assert.NotPanics(t, func() {
				got = EncodeKey(1, test.segments...)
			})

			assert.Equal(t, test.wantKey, got)
		})
	}
}
