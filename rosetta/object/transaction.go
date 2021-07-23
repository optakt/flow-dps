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

package object

import (
	"github.com/onflow/flow-go-sdk"

	"github.com/optakt/flow-dps/rosetta/identifier"
)

// Transaction contains an array of operations that are attributable to the same
// transaction identifier.
//
// Examples of metadata given in the Rosetta API documentation are "size" and
// "lockTime".
type Transaction struct {
	ID         identifier.Transaction `json:"transaction_identifier"`
	Operations []Operation            `json:"operations"`
}

// TransactionPayload is essentially a duplicate of flow.Transaction, only with proper JSON tags.
type TransactionPayload struct {
	Script             []byte                      `json:"script"`
	Arguments          [][]byte                    `json:"arguments"`
	ReferenceBlockID   flow.Identifier             `json:"reference_block_id"`
	GasLimit           uint64                      `json:"gas_limit"`
	ProposalKey        flow.ProposalKey            `json:"proposal_key"`
	Payer              flow.Address                `json:"payer"`
	Authorizers        []flow.Address              `json:"authorizers"`
	PayloadSignatures  []flow.TransactionSignature `json:"payload_signatures"`
	EnvelopeSignatures []flow.TransactionSignature `json:"envelope_signatures"`
}
