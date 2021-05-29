package lookup

import (
	"fmt"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/invoker"
)

func FromIndex(state dps.State) invoker.LookupFunc {
	return func(height uint64) (*flow.Header, flow.StateCommitment, error) {
		header, err := state.Data().Header(height)
		if err != nil {
			return nil, flow.StateCommitment{}, fmt.Errorf("could not retrieve header: %w", err)
		}
		commit, err := state.Commit().ForHeight(height)
		if err != nil {
			return nil, flow.StateCommitment{}, fmt.Errorf("could not retrieve commit: %w", err)
		}
		return header, commit, nil
	}
}
