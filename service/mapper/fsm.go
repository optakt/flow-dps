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
	"context"
	"fmt"
	"sync"
)

type edge struct {
	check      CheckFunc
	transition TransitionFunc
}

type FSM struct {
	state *State
	edges []edge
	wg    *sync.WaitGroup
}

func NewFSM(state *State) *FSM {

	f := &FSM{
		state: state,
		edges: []edge{},
		wg:    &sync.WaitGroup{},
	}

	return f
}

func (f *FSM) Add(check CheckFunc, transition TransitionFunc) {
	edge := edge{
		check:      check,
		transition: transition,
	}
	f.edges = append(f.edges, edge)
}

func (f *FSM) Run() error {
	f.wg.Add(1)
	defer f.wg.Done()
TransitionLoop:
	for {
		for _, e := range f.edges {
			ok := e.check(f.state)
			if !ok {
				continue
			}
			select {
			case <-f.state.done:
				break TransitionLoop
			default:
				// continue
			}
			err := e.transition(f.state)
			if err != nil {
				return fmt.Errorf("could not apply transition to state: %w", err)
			}
			continue TransitionLoop
		}
		return fmt.Errorf("could not find transition for state")
	}
	return nil
}

func (f *FSM) Stop(ctx context.Context) error {
	close(f.state.done)
	f.wg.Wait()
	return nil
}
