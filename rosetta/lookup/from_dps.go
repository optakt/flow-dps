package lookup

import (
	"context"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/rosetta/invoker"
)

func FromDPS(client dps.APIClient) invoker.LookupFunc {
	return func(height uint64) (*flow.Header, error) {
		req := dps.GetHeaderRequest{
			Height: &height,
		}
		res, err := client.GetHeader(context.Background(), &req)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve header: %w", err)
		}
		var header flow.Header
		err = cbor.Unmarshal(res.Data, &header)
		if err != nil {
			return nil, fmt.Errorf("could not decode header: %w", err)
		}
		return &header, nil
	}
}
