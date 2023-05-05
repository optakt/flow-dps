package mapper

import (
	"errors"
	"fmt"
	"sync"

	"github.com/onflow/flow-archive/models/archive"
	"github.com/rs/zerolog/log"
)

// FSM is a finite state machine which is used to map block data from multiple sources into
// the DPS index.
type FSM struct {
	state       *State
	transitions map[Status]TransitionFunc
	wg          *sync.WaitGroup
}

// NewFSM returns a new FSM using the given state and options.
func NewFSM(state *State, options ...func(*FSM)) *FSM {

	f := FSM{
		state:       state,
		transitions: make(map[Status]TransitionFunc),
		wg:          &sync.WaitGroup{},
	}

	for _, option := range options {
		option(&f)
	}

	return &f
}

// Run starts the state machine.
func (f *FSM) Run() error {
	f.wg.Add(1)
	defer f.wg.Done()

	for {
		select {
		case <-f.state.done:
			return nil
		default:
			// continue
		}

		transition, ok := f.transitions[f.state.status]
		if !ok {
			return fmt.Errorf("could not find transition for status (%d)", f.state.status)
		}

		startState := f.state.status.String()

		lg := log.With().Str("module", "FSM").Str("start_state", startState).Logger()

		lg.Info().Uint64("height", f.state.height).Msgf("before transition")

		err := transition(f.state)
		if errors.Is(err, archive.ErrFinished) {
			lg.Info().Msgf("finished state transition")
			return nil
		}
		if err != nil {
			lg.Error().Err(err).Msgf("failed state transition")
			return fmt.Errorf("could not apply transition to state: %w", err)
		}

		endState := f.state.status.String()

		lg.Info().Uint64("height", f.state.height).Str("end_state", endState).Msgf("after transition")
	}
}

// Stop gracefully stops the state machine.
func (f *FSM) Stop() error {
	close(f.state.done)
	f.wg.Wait()
	return nil
}
