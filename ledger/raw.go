package ledger

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
)

type Raw struct {
	core *Core
}

// Get returns the raw ledger data from the given ledger key, without the
// original key information.
func (r *Raw) Get(height uint64, key []byte) ([]byte, error) {

	payload, err := r.core.Payload(height, ledger.Path(key))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve payload: %w", err)
	}

	return payload.Value, nil
}
