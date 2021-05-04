package dps

import (
	"github.com/onflow/flow-go/ledger"
)

type Ledger interface {
	Payload(height uint64, key ledger.Key) (ledger.Payload, error)
	Get(query *ledger.Query) ([]ledger.Value, error)
}
