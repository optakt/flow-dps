package state

import (
	"github.com/onflow/flow-go/model/flow"
)

type Last struct {
	core *Core
}

func (l *Last) Height() uint64 {
	return l.core.height
}

func (l *Last) Commit() flow.StateCommitment {
	return l.core.commit
}
