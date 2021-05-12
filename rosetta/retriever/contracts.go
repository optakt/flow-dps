package retriever

import (
	"github.com/onflow/flow-go/model/flow"
)

type Contracts interface {
	Token(symbol string) (flow.Address, bool)
}
