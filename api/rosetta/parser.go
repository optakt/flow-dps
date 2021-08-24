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

package rosetta

import (
	"github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
	"github.com/optakt/flow-dps/rosetta/transactions"
)

// Parser is used by the Rosetta Construction API to handle transaction related operations.
type Parser interface {
	DeriveIntent(operations []object.Operation) (intent *transactions.Intent, err error)
	CompileTransaction(intent *transactions.Intent, metadata object.Metadata) (tx *flow.Transaction, err error)
	ParseTransaction(tx *flow.Transaction) (operations []object.Operation, signers []identifier.Account, err error)
	SignTransaction(unsignedTx *flow.Transaction, signature object.Signature) (tx *flow.Transaction, err error)
}
