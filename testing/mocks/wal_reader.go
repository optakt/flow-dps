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
