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

package response

import (
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/meta"
	"github.com/optakt/flow-dps/rosetta/object"
)

// Balance implements the successful response schema for /account/balance.
// See https://www.rosetta-api.org/docs/AccountApi.html#200---ok
type Balance struct {
	BlockID  identifier.Block `json:"block_identifier"`
	Balances []object.Amount  `json:"balances"`
}

// Block implements the response schema for /block.
// See https://www.rosetta-api.org/docs/BlockApi.html#200---ok
type Block struct {
	Block             *object.Block            `json:"block"`
	OtherTransactions []identifier.Transaction `json:"other_transactions,omitempty"`
}

// Combine implements the response schema for /construction/combine.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response
type Combine struct {
	SignedTransaction string `json:"signed_transaction"`
}

// Hash implements the response schema for /construction/hash.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-2
type Hash struct {
	TransactionID identifier.Transaction `json:"transaction_identifier"`
}

// Metadata implements the response schema for /construction/metadata.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-3
type Metadata struct {
	Metadata object.Metadata `json:"metadata"`
}

// Networks implements the successful response schema for the /network/list endpoint.
// See https://www.rosetta-api.org/docs/NetworkApi.html#200---ok
type Networks struct {
	NetworkIDs []identifier.Network `json:"network_identifiers"`
}

// Options implements the successful response schema for the /network/options endpoint.
// See https://www.rosetta-api.org/docs/NetworkApi.html#200---ok-1
type Options struct {
	Version meta.Version `json:"version"`
	Allow   OptionsAllow `json:"allow"`
}

// OptionsAllow specifies supported Operation statuses, Operation types, and all possible
// error statuses. It is returned by the /network/options endpoint.
type OptionsAllow struct {
	OperationStatuses       []meta.StatusDefinition `json:"operation_statuses"`
	OperationTypes          []string                `json:"operation_types"`
	Errors                  []meta.ErrorDefinition  `json:"errors"`
	HistoricalBalanceLookup bool                    `json:"historical_balance_lookup"`
	CallMethods             []string                `json:"call_methods"`       // not used
	BalanceExemptions       []struct{}              `json:"balance_exemptions"` // not used
	MempoolCoins            bool                    `json:"mempool_coins"`
}

// Parse implements the response schema for /construction/parse.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-4
type Parse struct {
	Operations []object.Operation   `json:"operations"`
	SignerIDs  []identifier.Account `json:"account_identifier_signers,omitempty"`
	Metadata   object.Metadata      `json:"metadata,omitempty"`
}

// Payloads implements the response schema for /construction/payloads.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-5
type Payloads struct {
	Transaction string                  `json:"unsigned_transaction"`
	Payloads    []object.SigningPayload `json:"payloads"`
}

// Preprocess implements the response schema for /construction/preprocess.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-6
type Preprocess struct {
	object.Options `json:"options,omitempty"`
}

// Status implements the successful response schema for /network/status.
// See https://www.rosetta-api.org/docs/NetworkApi.html#200---ok-2
type Status struct {
	CurrentBlockID        identifier.Block `json:"current_block_identifier"`
	CurrentBlockTimestamp int64            `json:"current_block_timestamp"`
	OldestBlockID         identifier.Block `json:"oldest_block_identifier"`
	GenesisBlockID        identifier.Block `json:"genesis_block_identifier"`
	Peers                 []struct{}       `json:"peers"` // not used
}

// Submit implements the response schema for /construction/submit.
// See https://www.rosetta-api.org/docs/ConstructionApi.html#response-7
type Submit struct {
	TransactionID identifier.Transaction `json:"transaction_identifier"`
}

// Transaction implements the successful response schema for /block/transaction.
// See https://www.rosetta-api.org/docs/BlockApi.html#200---ok-1
type Transaction struct {
	Transaction *object.Transaction `json:"transaction"`
}
