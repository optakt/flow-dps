package mapper

import (
	"github.com/awfm9/flow-dps/model"
	"github.com/onflow/flow-go/model/flow"
)

type Feeder interface {
	Feed(commit flow.StateCommitment) (model.Delta, error)
}
