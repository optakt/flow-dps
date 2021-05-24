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
	"github.com/optakt/flow-dps/models/rosetta"
)

type BlockResponse struct {
	Block             rosetta.Block            `json:"block"`
	OtherTransactions []identifier.Transaction `json:"other_transactions"`
}

type TransactionResponse struct {
	Transaction rosetta.Transaction `json:"transaction"`
}

type BalanceResponse struct {
	BlockID  identifier.Block `json:"block_identifier"`
	Balances []rosetta.Amount `json:"balances"`
}
