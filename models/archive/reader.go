package archive

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

// Reader represents something that can read from a DPS index.
type Reader interface {
	First() (uint64, error)
	Last() (uint64, error)

	HeightForBlock(blockID flow.Identifier) (uint64, error)
	HeightForTransaction(txID flow.Identifier) (uint64, error)

	Commit(height uint64) (flow.StateCommitment, error)
	Header(height uint64) (*flow.Header, error)
	Events(height uint64, types ...flow.EventType) ([]flow.Event, error)
	Values(height uint64, paths []ledger.Path) ([]ledger.Value, error)

	Collection(collID flow.Identifier) (*flow.LightCollection, error)
	Guarantee(collID flow.Identifier) (*flow.CollectionGuarantee, error)
	Transaction(txID flow.Identifier) (*flow.TransactionBody, error)
	Seal(sealID flow.Identifier) (*flow.Seal, error)
	Result(txID flow.Identifier) (*flow.TransactionResult, error)

	CollectionsByHeight(height uint64) ([]flow.Identifier, error)
	TransactionsByHeight(height uint64) ([]flow.Identifier, error)
	SealsByHeight(height uint64) ([]flow.Identifier, error)
}
