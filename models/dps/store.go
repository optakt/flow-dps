package dps

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type Store interface {
	Save(hash hash.Hash, payload *ledger.Payload) error
	Retrieve(hash hash.Hash) (*ledger.Payload, error)
	Close() error
}
