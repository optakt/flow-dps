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

package archive

import (
	"github.com/onflow/flow-go/model/flow"
)

// Chain represents something that has access to chain data.
type Chain interface {
	Root() (uint64, error)
	Header(height uint64) (*flow.Header, error)
	Commit(height uint64) (flow.StateCommitment, error)
	Events(height uint64) ([]flow.Event, error)
	Collections(height uint64) ([]*flow.LightCollection, error)
	Guarantees(height uint64) ([]*flow.CollectionGuarantee, error)
	Transactions(height uint64) ([]*flow.TransactionBody, error)
	Results(height uint64) ([]*flow.TransactionResult, error)
	Seals(height uint64) ([]*flow.Seal, error)
}
