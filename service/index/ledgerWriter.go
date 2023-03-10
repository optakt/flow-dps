package index

import (
	"fmt"

	"github.com/onflow/flow-archive/ledgertmp"
	"github.com/onflow/flow-archive/service/storage"
)

// Registers writes the given registers in a batch to database
func (w *Writer) Registers(height uint64, registers []*ledgertmp.LeafNode) error {
	writeBatch := storage.NewBatch(w.db)

	for _, register := range registers {
		err := w.lib.BatchSavePayload(height, register.Path, register.Payload)(writeBatch)
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
