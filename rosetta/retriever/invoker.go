package retriever

import (
	"github.com/onflow/cadence"
)

type Invoker interface {
	Script(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error)
}
