package archive

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"
)

// Writer represents something that can write on a DPS index.
type Writer interface {
	First(height uint64) error
	Last(height uint64) error

	Height(blockID flow.Identifier, height uint64) error

	Commit(height uint64, commit flow.StateCommitment) error
	Header(height uint64, header *flow.Header) error
	Events(height uint64, events []flow.Event) error
	Payloads(height uint64, payloads []*ledger.Payload) error
	Registers(height uint64, registers []*wal.LeafNode) error

	Collections(height uint64, collections []*flow.LightCollection) error
	Guarantees(height uint64, guarantees []*flow.CollectionGuarantee) error
	Transactions(height uint64, transactions []*flow.TransactionBody) error
	Results(results []*flow.TransactionResult) error
	Seals(height uint64, seals []*flow.Seal) error
}
