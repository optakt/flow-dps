package storage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-archive/testing/mocks"
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
