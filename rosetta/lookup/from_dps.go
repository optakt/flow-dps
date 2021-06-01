package lookup

import (
	"fmt"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/rosetta/invoker"
)

func FromDPS(_ dps.APIClient) invoker.LookupFunc {
	return func(height uint64) (*flow.Header, flow.StateCommitment, error) {
		// TODO: implement RPC for header and commit retrieval and use here
		return nil, flow.StateCommitment{}, fmt.Errorf("not implemented")
	}
}
