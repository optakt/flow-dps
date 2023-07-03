package access

import (
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"
)

// Invoker represents something that can retrieve accounts at any given height, and execute scripts to retrieve values
// from the Flow Virtual Machine.
type Invoker interface {
	Account(height uint64, address flow.Address) (*flow.Account, error)
	Script(height uint64, script []byte, parameters [][]byte) (cadence.Value, error)
}
