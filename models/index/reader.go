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

package index

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Reader interface {
	First() (uint64, error)
	Last() (uint64, error)
	Header(height uint64) (*flow.Header, error)
	Commit(height uint64) (flow.StateCommitment, error)
	Events(height uint64, types ...flow.EventType) ([]flow.Event, error)
	Registers(height uint64, paths []ledger.Path) ([]ledger.Value, error)
	Height(blockID flow.Identifier) (uint64, error)
	Transaction(transactionID flow.Identifier) (*flow.Transaction, error)
	Transactions(blockID flow.Identifier) ([]flow.Identifier, error)
}
