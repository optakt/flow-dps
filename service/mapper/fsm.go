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
)

type Edge struct {
	check      CheckFunc
	transition TransitionFunc
}

type FSM struct {
	edges []Edge
}

func NewFSM() *FSM {

	f := &FSM{}

	return f
}

func (f *FSM) Add(check CheckFunc, transition TransitionFunc) {
	edge := Edge{
		check:      check,
		transition: transition,
	}
	f.edges = append(f.edges, edge)
}

func (f *FSM) Run(s *State) error {
TransitionLoop:
	for {
		for _, edge := range f.edges {
			ok := edge.check(s)
			if !ok {
				continue
			}
			err := edge.transition(s)
			if err != nil {
				return fmt.Errorf("could not apply transition to state: %w", err)
			}
			continue TransitionLoop
		}
		return fmt.Errorf("could not find transition for state")
	}
}

func (f *FSM) Stop(ctx context.Context) error {
	return nil
}
