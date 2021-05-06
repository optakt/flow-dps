package rest

import (
	"github.com/onflow/flow-go/ledger"
)

type Raw interface {
	Get(height uint64, key []byte) ([]byte, error)
}

type Ledger interface {
	Get(*ledger.Query) (*ledger.Payload, error)
}
