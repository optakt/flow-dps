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

const (
	FlowSignatureAlgorithm = "ecdsa"
)

// TODO: We could consider moving Data and Construction APIs to separate packages.
// TODO: when the construction API crystalizes, make all the error returns consistent
// TODO: as a WIP thing - treat the sender as the proposer and the payer. Consider allowing more granular control.
// TODO: set gas limit for the transaction.
// TODO: get account sequence number for the metadata endpoint
//
// TODO: check - We are serialiing/deserializing the transactions a few times. Originally
// I created a TransactionPayload structure that is identical to flow.Transaction, except
// it had JSON tags so the outputted JSON was nicer. However, since it led to overhead
// and the need to convert the structures back and forth, I opted for removing it and always
// using flow.Transaction, even if the resulting JSON reads like `{ "Script": "..." }` (uppercased field name).

// Construction implements the Rosetta Construction API specification.
// See https://www.rosetta-api.org/docs/construction_api_introduction.html
type Construction struct {
	config    Configuration
	parser    Parser
	retriever Retriever
	client    Client
}

// NewConstruction creates a new instance of the Construction API using the given configuration
// to handle transaction construction requests.
func NewConstruction(config Configuration, parser Parser, retriever Retriever, client Client) *Construction {

	// The retriever has a number of capabilities, but in the context of the
	// Rosetta Construction API, it is only used to retrieve the latest block ID.
	// This is needed since the transactions require a reference block ID, so that
	// their validity or expiration can be determined.

	// Client is used to submit the constructed transaction to the Flow network.

	c := Construction{
		config:    config,
		parser:    parser,
		retriever: retriever,
		client:    client,
	}

	return &c
}
