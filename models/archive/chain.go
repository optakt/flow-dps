package archive

import (
	"github.com/onflow/flow-go/model/flow"
)

// Chain represents something that has access to chain data.
type Chain interface {
	Root() (uint64, error)
	Header(height uint64) (*flow.Header, error)
	Commit(height uint64) (flow.StateCommitment, error)
	Events(height uint64) ([]flow.Event, error)
	Collections(height uint64) ([]*flow.LightCollection, error)
	Guarantees(height uint64) ([]*flow.CollectionGuarantee, error)
	Transactions(height uint64) ([]*flow.TransactionBody, error)
	Results(height uint64) ([]*flow.TransactionResult, error)
	Seals(height uint64) ([]*flow.Seal, error)
}
