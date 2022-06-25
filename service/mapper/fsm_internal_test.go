// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package mapper

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-dps/testing/mocks"
)

func TestNewFSM(t *testing.T) {
	st := &State{}

	t.Run("nominal case without options", func(t *testing.T) {
		t.Parallel()

		f := NewFSM(st)

		assert.NotNil(t, f)
		assert.Equal(t, st, f.state)
	})

	t.Run("nominal case with transition option", func(t *testing.T) {
		t.Parallel()

		f := NewFSM(st, WithTransition(StatusBootstrap, func(*State) error { return nil }))

		assert.NotNil(t, f)
		assert.Equal(t, st, f.state)
		assert.Len(t, f.transitions, 1)
	})
}

func TestFSM_Run(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		var emptyCalls, matchedCalls int
		f := &FSM{
			state: &State{
				status: StatusBootstrap,
				done:   make(chan struct{}),
			},
			transitions: map[Status]TransitionFunc{
				StatusBootstrap: func(state *State) error {
					state.status = StatusCollect
					emptyCalls++
					return nil
				},
				StatusCollect: func(state *State) error {
					state.status = StatusBootstrap
					matchedCalls++
					return nil
				},
			},
			wg: &sync.WaitGroup{},
		}

		done := make(chan struct{})
		go func() {
			err := f.Run()

			assert.NoError(t, err)

			close(done)
		}()

		// Make run stop after running for a few loops.
		time.Sleep(5 * time.Millisecond)
		close(f.state.done)

		// Wait for it to return successfully and let us do assertions before stopping test.
		<-done

		assert.NotZero(t, emptyCalls)
		assert.NotZero(t, matchedCalls)
	})

	t.Run("transition does not exist for given state", func(t *testing.T) {
		t.Parallel()

		f := &FSM{
			state: &State{
				status: StatusBootstrap,
			},
			transitions: map[Status]TransitionFunc{
				StatusForward: func(*State) error { return nil },
			},
			wg: &sync.WaitGroup{},
		}

		err := f.Run()

		assert.Error(t, err)
	})

	t.Run("transition fails", func(t *testing.T) {
		t.Parallel()

		failingTransition := func(*State) error { return mocks.GenericError }

		f := &FSM{
			state: &State{
				status: StatusBootstrap,
			},
			transitions: map[Status]TransitionFunc{
				StatusBootstrap: failingTransition,
			},
			wg: &sync.WaitGroup{},
		}

		err := f.Run()

		assert.Error(t, err)
	})
}

func TestFSM_Stop(t *testing.T) {
	f := &FSM{
		state: &State{
			done: make(chan struct{}),
		},
		wg: &sync.WaitGroup{},
	}

	err := f.Stop()

	assert.NoError(t, err)
}
