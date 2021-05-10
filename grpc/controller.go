package grpc

import (
	"context"
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"google.golang.org/grpc"

	"github.com/awfm9/flow-dps/model"
)

type Controller struct {
	state model.State
}

func NewController(state model.State) (*Controller, error) {
	c := &Controller{
		state: state,
	}
	return c, nil
}

func (c *Controller) GetRegister(_ context.Context, req *GetRegisterRequest, _ ...grpc.CallOption) (*Register, error) {
	state := c.state.Raw()

	height, _ := c.state.Last()
	if req.Height != nil {
		height = *req.Height
	}

	state = state.WithHeight(height)

	value, err := state.Get(req.Key)
	if err != nil {
		return nil, fmt.Errorf("could not get register in GRPC API: %w", err)
	}

	return &Register{
		Height: height,
		Key: req.Key,
		Value: value,
	}, nil
}

// GetValues returns the payload value of an encoded Ledger entry in the same way
// as the Flow Ledger interface would. It takes an input that emulates the `ledger.Query` struct.
// The state hash and the pathfinder key version are optional as part of the request.
// If omitted, the state hash of the latest sealed block and the default pathfinder key encoding is used.
func (c *Controller) GetValues(_ context.Context, req *GetValuesRequest, _ ...grpc.CallOption) (*Values, error) {
	state := c.state.Ledger()

	if req.Version != nil {
		state = state.WithVersion(uint8(*req.Version))
	}

	_, commit := c.state.Last()
	if req.Hash != nil {
		commit = req.Hash
	}

	var keys []ledger.Key
	for _, key := range req.Keys {
		var k ledger.Key
		for _, part := range key.Parts {
			k.KeyParts = append(k.KeyParts, ledger.NewKeyPart(uint16(part.Type), part.Value))
		}
		keys = append(keys, k)
	}

	query, err := ledger.NewQuery(commit, keys)
	if err != nil {
		return nil, fmt.Errorf("could not forge query in GRPC API: %w", err)
	}

	values, err := state.Get(query)
	if err != nil {
		return nil, fmt.Errorf("could not get values in GRPC API: %w", err)
	}

	// Convert the ledger.Values into [][]byte.
	var v [][]byte
	for _, value := range values {
		v = append(v, value)
	}

	return &Values{
		Values: v,
	}, nil
}
