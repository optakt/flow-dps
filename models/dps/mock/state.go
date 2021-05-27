package mock

import (
	"github.com/stretchr/testify/mock"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type State struct {
	mock.Mock

	IndexState  *Index
	ChainState  *Chain
	LastState   *Last
	HeightState *Height
	CommitState *Commit
	RawState    *Raw
	LedgerState *Ledger
}

func NewState() *State {
	state := State{
		IndexState:  &Index{},
		ChainState:  &Chain{},
		LastState:   &Last{},
		HeightState: &Height{},
		CommitState: &Commit{},
		RawState:    &Raw{},
		LedgerState: &Ledger{},
	}

	return &state
}

func (s *State) Index() dps.Index {
	return s.IndexState
}

func (s *State) Chain() dps.Chain {
	return s.ChainState
}

func (s *State) Last() dps.Last {
	return s.LastState
}

func (s *State) Height() dps.Height {
	return s.HeightState
}

func (s *State) Commit() dps.Commit {
	return s.CommitState
}

func (s *State) Raw() dps.Raw {
	return s.RawState
}

func (s *State) Ledger() dps.Ledger {
	return s.LedgerState
}

type Index struct {
	mock.Mock
}

func (i *Index) Header(height uint64, header *flow.Header) error {
	args := i.Called(height, header)
	return args.Error(0)
}

func (i *Index) Commit(height uint64, commit flow.StateCommitment) error {
	args := i.Called(height, commit)
	return args.Error(0)
}

func (i *Index) Payloads(height uint64, paths []ledger.Path, payload []*ledger.Payload) error {
	args := i.Called(height, paths, payload)
	return args.Error(0)
}

func (i *Index) Events(height uint64, events []flow.Event) error {
	args := i.Called(height, events)
	return args.Error(0)
}

func (i *Index) Last(commit flow.StateCommitment) error {
	args := i.Called(commit)
	return args.Error(0)
}

type Chain struct {
	mock.Mock
}

func (c *Chain) Header(height uint64) (*flow.Header, error) {
	args := c.Called(height)
	return args.Get(0).(*flow.Header), args.Error(1)
}

func (c *Chain) Events(height uint64, types ...flow.EventType) ([]flow.Event, error) {
	args := c.Called(height, types)
	return args.Get(0).([]flow.Event), args.Error(1)
}

type Last struct {
	mock.Mock
}

func (l *Last) Height() uint64 {
	args := l.Called()
	return args.Get(0).(uint64)
}

func (l *Last) Commit() flow.StateCommitment {
	args := l.Called()
	return args.Get(0).(flow.StateCommitment)
}

type Height struct {
	mock.Mock
}

func (h *Height) ForBlock(blockID flow.Identifier) (uint64, error) {
	args := h.Called(blockID)
	return args.Get(0).(uint64), args.Error(1)
}

func (h *Height) ForCommit(commit flow.StateCommitment) (uint64, error) {
	args := h.Called(commit)
	return args.Get(0).(uint64), args.Error(1)
}

type Commit struct {
	mock.Mock
}

func (c *Commit) ForHeight(height uint64) (flow.StateCommitment, error) {
	args := c.Called(height)
	return args.Get(0).(flow.StateCommitment), args.Error(1)
}

type Raw struct {
	mock.Mock
}

func (r *Raw) WithHeight(height uint64) dps.Raw {
	args := r.Called(height)
	return args.Get(0).(dps.Raw)
}

func (r *Raw) Get(key []byte) ([]byte, error) {
	args := r.Called(key)
	return args.Get(0).([]byte), args.Error(1)
}

type Ledger struct {
	mock.Mock
}

func (l *Ledger) WithVersion(version uint8) dps.Ledger {
	args := l.Called(version)
	return args.Get(0).(dps.Ledger)
}

func (l *Ledger) Get(query *ledger.Query) ([]ledger.Value, error) {
	args := l.Called(query)
	return args.Get(0).([]ledger.Value), args.Error(1)
}
