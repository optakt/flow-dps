package ledger

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
)

type Raw struct {
	core   *Core
	height uint64
}

func (r *Raw) WithHeight(height uint64) *Raw {
	r.height = height
	return r
}

// Get returns the raw ledger data from the given ledger key, without the
// original key information.
func (r *Raw) Get(key []byte) ([]byte, error) {

	payload, err := r.core.Payload(r.height, ledger.Path(key))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve payload: %w", err)
	}

	return payload.Value, nil
}
