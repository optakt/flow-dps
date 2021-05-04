package mapper

import (
	"github.com/onflow/flow-go/model/flow"
)

type Chain interface {
	Active() (uint64, flow.Identifier, flow.StateCommitment)
	Forward() error
}
