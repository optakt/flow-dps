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

package dps

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

// Writer represents something that can write on a DPS index.
type Writer interface {
	First(height uint64) error
	Last(height uint64) error

	Height(blockID flow.Identifier, height uint64) error

	Commit(height uint64, commit flow.StateCommitment) error
	Header(height uint64, header *flow.Header) error
	Events(height uint64, events []flow.Event) error
	Payloads(height uint64, paths []ledger.Path, values []*ledger.Payload) error

	Collections(height uint64, collections []*flow.LightCollection) error
	Guarantees(height uint64, guarantees []*flow.CollectionGuarantee) error
	Transactions(height uint64, transactions []*flow.TransactionBody) error
	Results(results []*flow.TransactionResult) error
	Seals(height uint64, seals []*flow.Seal) error

	Close() error
}
