package retriever

import (
	"github.com/onflow/flow-go/model/flow"
)

type Scripts interface {
	GetBalance(token flow.Address) []byte
}
