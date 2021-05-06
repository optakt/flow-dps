package rest

import (
	"github.com/onflow/flow-go/ledger"
)

type State interface {
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
