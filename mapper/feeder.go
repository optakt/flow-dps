package mapper

import (
	"github.com/onflow/flow-go/ledger"
)

type Feeder interface {
	Feed() (*ledger.TrieUpdate, error)
}
