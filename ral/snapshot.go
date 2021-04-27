package ral

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
)

type Snapshot struct {
	core    *Core
	version uint8
	height  uint64
}

// Get implements the get method of the Flow ledger interface.
func (s *Snapshot) Get(query *ledger.Query) ([]ledger.Value, error) {
	paths, err := pathfinder.KeysToPaths(query.Keys(), s.version)
	if err != nil {
		return nil, fmt.Errorf("could not convert keys to paths (%w)", err)
	}
	payloads := make([]*ledger.Payload, 0, len(paths))
	for _, path := range paths {
		payload, err := s.core.Payload(s.height, path)
		if err != nil {
			return nil, fmt.Errorf("could not get payload (%w)", err)
		}
		payloads = append(payloads, payload)
	}
	values, err := pathfinder.PayloadsToValues(payloads)
	if err != nil {
		return nil, fmt.Errorf("could not convert payloads to values (%w)", err)
	}
	return values, nil
}

// Set implements the set method of the Flow ledger interface.
func (s *Snapshot) Set(update *ledger.Update) (ledger.State, error) {
	return nil, fmt.Errorf("the DPS ledger is read-only")
}

// Prove implements the prove method of the Flow ledger interface.
func (s *Snapshot) Prove(query *ledger.Query) (ledger.Proof, error) {
	return nil, fmt.Errorf("proofs are not implemented yet")
}
