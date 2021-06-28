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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

func TestEncodeKey(t *testing.T) {
	id, _ := flow.HexStringToIdentifier("aac513eb1a0457700ac3fa8d292513e18ad7fd70065146b35ab48fa5a6cab007")
	path := ledger.Path{0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x04, 0x57, 0x70, 0x0a, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1}
	commit, _ := flow.ToStateCommitment([]byte("07018030187ecf04945f35f1e33a89dc"))

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
			wantKey: []byte{
				0x1,                                     // prefix
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2a, // uint(42)
				0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x4, 0x57, 0x70, 0xa, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1, 0x8a, 0xd7, 0xfd, 0x70, 0x6, 0x51, 0x46, 0xb3, 0x5a, 0xb4, 0x8f, 0xa5, 0xa6, 0xca, 0xb0, 0x7, // id
				0xaa, 0xc5, 0x13, 0xeb, 0x1a, 0x4, 0x57, 0x70, 0xa, 0xc3, 0xfa, 0x8d, 0x29, 0x25, 0x13, 0xe1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x30, 0x37, 0x30, 0x31, // path
				0x38, 0x30, 0x33, 0x30, 0x31, 0x38, 0x37, 0x65, 0x63, 0x66, 0x30, 0x34, 0x39, 0x34, 0x35, 0x66, 0x33, 0x35, 0x66, 0x31, 0x65, 0x33, 0x33, 0x61, 0x38, 0x39, 0x64, 0x63, // commit
			},
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
					encodeKey(1, test.segments...)
				})
				return
			}

			var got []byte
			assert.NotPanics(t, func() {
				got = encodeKey(1, test.segments...)
			})

			assert.Equal(t, test.wantKey, got)
		})
	}
}
