package index

import (
	"fmt"

	"github.com/onflow/flow-archive/util"
	"github.com/onflow/flow-go/ledger/complete/wal"
)

// Registers writes the given registers in a batch to database
func (w *Writer) Registers(height uint64, registers []*wal.LeafNode) error {
	writeBatch := util.NewBatch(w.db)

	for _, register := range registers {
		op := w.lib.BatchSavePayload(height, register.Path, register.Payload)
		err := op(writeBatch)
		if err != nil {
			return fmt.Errorf("could not batch write registers to database at height %v: %w", height, err)
		}
	}

	err := writeBatch.Flush()
	if err != nil {
		return fmt.Errorf("could not flush write registers to database at height %v: %w", height, err)
	}

	return nil
}
