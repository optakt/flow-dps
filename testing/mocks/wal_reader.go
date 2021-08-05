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

package mocks

import (
	"testing"
)

type WALReader struct {
	NextFunc   func() bool
	ErrFunc    func() error
	RecordFunc func() []byte
}

func BaselineWALReader(t *testing.T) *WALReader {
	t.Helper()

	return &WALReader{
		NextFunc: func() bool {
			return true
		},
		ErrFunc: func() error {
			return nil
		},
		RecordFunc: func() []byte {
			return GenericBytes
		},
	}
}

func (w *WALReader) Next() bool {
	return w.NextFunc()
}

func (w *WALReader) Err() error {
	return w.ErrFunc()
}

func (w *WALReader) Record() []byte {
	return w.RecordFunc()
}
