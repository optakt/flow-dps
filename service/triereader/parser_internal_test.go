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

package triereader

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"

	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-archive/testing/mocks"
)

func TestFromWAL(t *testing.T) {
	reader := mocks.BaselineWALReader(t)

	feeder := FromWAL(reader)

	assert.Equal(t, reader, feeder.reader)
}

func TestFeeder_Update(t *testing.T) {
	update := mocks.GenericTrieUpdate(0)
	data := ledger.EncodeTrieUpdate(update)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var recordCalled bool
		reader := mocks.BaselineWALReader(t)
		reader.RecordFunc = func() []byte {
			builder := strings.Builder{}

			// On the first call, return a Delete operation which should get ignored and skipped.
			if !recordCalled {
				recordCalled = true
				size := len(update.RootHash)
				buffer := make([]byte, 2)
				binary.BigEndian.PutUint16(buffer, uint16(size))

				_ = builder.WriteByte(byte(wal.WALDelete))
				_, _ = builder.Write(buffer)
				_, _ = builder.Write(update.RootHash[:])

				return []byte(builder.String())
			}

			// On any subsequent call, return the Update operation.
			_ = builder.WriteByte(byte(wal.WALUpdate))
			_, _ = builder.Write(data[:])

			return []byte(builder.String())
		}

		updates := &WalParser{
			reader: reader,
		}

		got, err := updates.AllUpdates()

		require.NoError(t, err)
		assert.Equal(t, update, got)
	})

	t.Run("handles reader failure", func(t *testing.T) {
		t.Parallel()

		reader := mocks.BaselineWALReader(t)
		reader.NextFunc = func() bool {
			return false
		}
		reader.ErrFunc = func() error {
			return mocks.GenericError
		}

		updates := &WalParser{
			reader: reader,
		}

		_, err := updates.AllUpdates()

		assert.Error(t, err)
	})

	t.Run("handles unavailable next record", func(t *testing.T) {
		t.Parallel()

		reader := mocks.BaselineWALReader(t)
		reader.NextFunc = func() bool {
			return false
		}

		updates := &WalParser{
			reader: reader,
		}

		_, err := updates.AllUpdates()

		require.Error(t, err)
		assert.Equal(t, archive.ErrUnavailable, err)
	})

	t.Run("handles badly encoded record", func(t *testing.T) {
		t.Parallel()

		reader := mocks.BaselineWALReader(t)
		reader.RecordFunc = func() []byte {
			return mocks.GenericBytes
		}

		updates := &WalParser{
			reader: reader,
		}

		_, err := updates.AllUpdates()

		assert.Error(t, err)
	})
}
