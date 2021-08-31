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
	sdk "github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
	"github.com/optakt/flow-dps/rosetta/transactor"
)

// Transactor is used by the Rosetta Construction API to handle transaction related operations.
type Transactor interface {
	DeriveIntent(operations []object.Operation) (intent *transactor.Intent, err error)
	CompileTransaction(refBlockID identifier.Block, intent *transactor.Intent, sequence uint64) (tx *sdk.Transaction, err error)
	HashPayload(rosBlockID identifier.Block, tx *sdk.Transaction, signer identifier.Account) (payload []byte, err error)
	ParseTransaction(tx *sdk.Transaction) (operations []object.Operation, signers []identifier.Account, err error)
	AttachSignature(unsignedTx *sdk.Transaction, signature object.Signature) (tx *sdk.Transaction, err error)
}
