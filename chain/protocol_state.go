package chain

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/onflow/flow-go/model/flow"
)

type ProtocolState struct {
	state *badger.DB
}

func FromProtocolState(dir string) (*ProtocolState, error) {

	opts := badger.DefaultOptions(dir)
	state, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not open badger database: %w", err)
	}

	ps := &ProtocolState{
		state: state,
	}

	return ps, nil
}

func (ps *ProtocolState) Active() (uint64, flow.Identifier, flow.StateCommitment) {
	return 0, flow.ZeroID, flow.StateCommitment{}
}

func (ps *ProtocolState) Forward() error {
	return nil
}
