package rest

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type State interface {
	Last() (uint64, flow.StateCommitment)
	Raw() Raw
	Ledger() Ledger
}

type Raw interface {
	WithHeight(height uint64) Raw
	Get(key []byte) ([]byte, error)
}

type Ledger interface {
	WithVersion(version uint8) Ledger
	Get(*ledger.Query) ([]ledger.Value, error)
}
