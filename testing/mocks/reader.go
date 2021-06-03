// Copyright 2021 Alvalor S.A.
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

package mocks

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Reader struct {
	LastFunc      func() (uint64, error)
	HeaderFunc    func(height uint64) (*flow.Header, error)
	CommitFunc    func(height uint64) (flow.StateCommitment, error)
	EventsFunc    func(height uint64, types ...flow.EventType) ([]flow.Event, error)
	RegistersFunc func(height uint64, paths []ledger.Path) ([]ledger.Value, error)
}

func (r *Reader) Last() (uint64, error) {
	return r.LastFunc()
}

func (r *Reader) Header(height uint64) (*flow.Header, error) {
	return r.HeaderFunc(height)
}

func (r *Reader) Commit(height uint64) (flow.StateCommitment, error) {
	return r.CommitFunc(height)
}

func (r *Reader) Events(height uint64, types ...flow.EventType) ([]flow.Event, error) {
	return r.EventsFunc(height, types...)
}

func (r *Reader) Registers(height uint64, paths []ledger.Path) ([]ledger.Value, error) {
	return r.RegistersFunc(height, paths)
}
