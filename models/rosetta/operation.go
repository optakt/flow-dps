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

package rosetta

import (
	"github.com/optakt/flow-dps/models/identifier"
)

// Operation contains all balance-changing information within a transaction. It
// is always one-sided (only affects one account identifier) and can succeed or
// fail independently from a transaction. Operations are used both to represent
// on-chain data in the Data API and to construct new transaction in the
// Construction API, creating a standard interface for reading and writing to
// blockchains.
//
// Examples of metadata given in the Rosetta API documentation are
// "asm" and "hex".
//
// The `coin_change` field is ommitted, as the Flow blockchain is an
// account-based blockchain without utxo set.
type Operation struct {
	ID         identifier.Operation   `json:"operation_identifier"`
	RelatedIDs []identifier.Operation `json:"related_operations"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"`
	AccountID  identifier.Account     `json:"account"`
	Amount     Amount                 `json:"amount"`
}
