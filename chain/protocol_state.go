package chain

import (
	"errors"
	"fmt"

	"github.com/awfm9/flow-dps/model"
	"github.com/dgraph-io/badger/v2"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"
)

type ProtocolState struct {
	state   *badger.DB
	height  uint64
	blockID flow.Identifier
	commit  flow.StateCommitment
}

func FromProtocolState(dir string) (*ProtocolState, error) {

	opts := badger.DefaultOptions(dir).WithLogger(nil)
	state, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not open badger database: %w", err)
	}

	var height uint64
	err = operation.RetrieveRootHeight(&height)(state.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve root height: %w", err)
	}
	var blockID flow.Identifier
	err = operation.LookupBlockHeight(height, &blockID)(state.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not look up root block: %w", err)
	}
	var sealID flow.Identifier
	err = operation.LookupBlockSeal(blockID, &sealID)(state.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not look up root seal: %w", err)
	}
	var seal flow.Seal
	err = operation.RetrieveSeal(sealID, &seal)(state.NewTransaction(false))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve root seal: %w", err)
	}

	ps := &ProtocolState{
		state:   state,
		height:  height,
		blockID: blockID,
		commit:  seal.FinalState,
	}

	return ps, nil
}

func (ps *ProtocolState) Active() (uint64, flow.Identifier, flow.StateCommitment) {
	return ps.height, ps.blockID, ps.commit
}

func (ps *ProtocolState) Forward() error {
	height := ps.height + 1
	var blockID flow.Identifier
	err := operation.LookupBlockHeight(height, &blockID)(ps.state.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return model.ErrFinished
	}
	if err != nil {
		return fmt.Errorf("could not look up next block: %w", err)
	}
	var sealID flow.Identifier
	err = operation.LookupBlockSeal(blockID, &sealID)(ps.state.NewTransaction(false))
	if errors.Is(err, storage.ErrNotFound) {
		return model.ErrFinished
	}
	if err != nil {
		return fmt.Errorf("could not look up next seal: %w", err)
	}
	var seal flow.Seal
	err = operation.RetrieveSeal(sealID, &seal)(ps.state.NewTransaction(false))
	if err != nil {
		return fmt.Errorf("could not retrieve next seal: %w", err)
	}
	ps.height = height
	ps.blockID = blockID
	ps.commit = seal.FinalState
	return nil
}
