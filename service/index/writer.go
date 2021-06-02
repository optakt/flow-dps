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

package index

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Writer struct {
	db *badger.DB
}

func NewWriter(db *badger.DB) *Writer {

	w := Writer{
		db: db,
	}

	return &w
}

func (w *Writer) Header(height uint64, header *flow.Header) error {
	return fmt.Errorf("not implemented")
}

func (w *Writer) Commit(height uint64, commit flow.StateCommitment) error {
	return fmt.Errorf("not implemented")
}

func (w *Writer) Events(height uint64, events []flow.Event) error {
	return fmt.Errorf("not implemented")
}

func (w *Writer) Register(height uint64, path ledger.Path, value ledger.Value) error {
	return fmt.Errorf("not implemented")
}
