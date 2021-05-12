package state

import (
	"github.com/onflow/flow-go/model/flow"
)

type Chain struct {
	core *Core
}

func (c *Chain) Header(height uint64) (*flow.Header, error) {

	return nil, nil
}
