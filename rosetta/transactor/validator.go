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

package transactor

import (
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/rosetta/identifier"
)

// Validator represents something that can validate account and block identifiers as well as currencies.
type Validator interface {
	Account(rosAccountID identifier.Account) (address flow.Address, err error)
	Block(rosBlockID identifier.Block) (height uint64, blockID flow.Identifier, err error)
	Currency(currency identifier.Currency) (symbol string, decimals uint, err error)
}
